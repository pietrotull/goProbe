package hosts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/els0r/goProbe/pkg/goprobe/client"
	"github.com/els0r/goProbe/pkg/logging"
	"github.com/els0r/goProbe/pkg/query"
	"github.com/els0r/goProbe/pkg/results"
	"github.com/els0r/goProbe/pkg/types"
	"gopkg.in/yaml.v3"
)

type QuerierType string

const (
	APIClientQuerierType QuerierType = "api"
)

type Querier interface {
	CreateQueryWorkload(ctx context.Context, host string, args *query.Args) (*QueryWorkload, error)
}

type QueryWorkload struct {
	Host string

	Runner query.Runner
	Args   *query.Args

	Result *results.Result
	Err    error
}

type APIClientQuerier struct {
	apiEndpoints map[string]*client.Config
}

func NewAPIClientQuerier(cfgPath string) (*APIClientQuerier, error) {
	a := &APIClientQuerier{
		apiEndpoints: make(map[string]*client.Config),
	}

	// read in the endpoints config
	f, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(a.apiEndpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to read in config: %w", err)
	}
	return a, nil
}

func (a *APIClientQuerier) CreateQueryWorkload(ctx context.Context, host string, args *query.Args) (*QueryWorkload, error) {
	qw := &QueryWorkload{
		Host: host,
		Args: args,
	}
	// create the api client runner by looking up the endpoint config for the given host
	cfg, exists := a.apiEndpoints[host]
	if !exists {
		return nil, fmt.Errorf("couldn't find endpoint configuration for host")
	}
	qw.Runner = client.New().FromConfig(cfg)

	return qw, nil
}

// PrepareQueries creates query workloads for all hosts in the host list and returns the channel it sends the
// workloads on
func PrepareQueries(ctx context.Context, querier Querier, hostList Hosts, args *query.Args) <-chan *QueryWorkload {
	workloads := make(chan *QueryWorkload)

	go func(ctx context.Context) {
		logger := logging.WithContext(ctx)

		for _, host := range hostList {
			wl, err := querier.CreateQueryWorkload(ctx, host, args)
			if err != nil {
				logger.With("host", host).Errorf("failed to create workload: %v", err)
				continue
			}
			workloads <- wl
		}
		close(workloads)
	}(ctx)

	return workloads
}

var (
	ErrorNoDataReturned = errors.New("no data returned")
)

// RunQueries takes query workloads from the workloads channel, runs them, and returns a channel from which
// the results can be read
func RunQueries(ctx context.Context, numRunners int, workloads <-chan *QueryWorkload) <-chan *QueryWorkload {
	out := make(chan *QueryWorkload, numRunners)

	wg := new(sync.WaitGroup)
	wg.Add(numRunners)
	for i := 0; i < numRunners; i++ {
		go func(ctx context.Context) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case wl, open := <-workloads:
					if !open {
						return
					}

					ctx = logging.NewContext(ctx, "hostname", wl.Host)

					logger := logging.WithContext(ctx)
					logger.Debugf("running query")

					res, err := wl.Runner.Run(ctx, wl.Args)
					if err != nil {
						err = fmt.Errorf("failed to run query: %w", err)
					}
					wl.Result = res
					wl.Err = err

					out <- wl
				}
			}
		}(ctx)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// AggregateResults takes finished query workloads from the workloads channel, aggregates the result by merging the rows and summaries,
// and returns the final result. The `tracker` variable provides information about potential Run failures for individual hosts
func AggregateResults(ctx context.Context, stmt *query.Statement, workloads <-chan *QueryWorkload) (finalResult *results.Result, tracker results.HostsStatuses) {
	// aggregation
	finalResult = &results.Result{
		Status: results.Status{
			Code: types.StatusOK,
		},
	}
	tracker = make(results.HostsStatuses)

	var rowMap results.RowsMap

	logger := logging.WithContext(ctx)

	defer func() {
		if len(finalResult.Rows) > 0 {
			finalResult.Rows = rowMap.ToRowsSorted(results.By(stmt.SortBy, stmt.Direction, stmt.SortAscending))
		} else {
			finalResult.Status.Code = types.StatusEmpty
			finalResult.Status.Message = results.ErrorNoResults.Error()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case hq, open := <-workloads:
			if !open {
				return
			}
			logger := logger.With("hostname", hq.Host)
			if hq.Err != nil {
				tracker[hq.Host] = results.Status{
					Code:    types.StatusError,
					Message: hq.Err.Error(),
				}

				logger.Error(hq.Err)
				continue
			}

			res := hq.Result
			tracker[hq.Host] = res.Status

			rowMap.MergeRows(res.Rows)

			finalResult.Summary.Totals.Add(res.Summary.Totals)
			finalResult.Summary.Hits.Total = len(rowMap)
		}
	}
}
