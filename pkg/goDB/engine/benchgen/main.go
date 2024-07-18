package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/els0r/goProbe/pkg/query"
)

var (
	benchmarkBase = `// Code generated by benchgen/main.go; DO NOT EDIT
package engine

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/els0r/telemetry/logging"
)

// Pre-defined Benchmarks
// The filesystem cache is flushed after every run of the queryto ensure that
// I/O is properly accounted for

func BenchmarkStdQueryJSONOutput(b *testing.B) {

	buf := &bytes.Buffer{}

	benchQuery(b, buf, flushCaches,
		"eth1", "time",
		[]query.Option{
			query.WithFirst("0"),
			query.WithNumResults(query.MaxResults),
			query.WithFormat(types.FormatJSON),
		}...,
	)

	flushCaches()

	_ = buf
}

func BenchmarkStdQueryJSONOutputCondition(b *testing.B) {

	buf := &bytes.Buffer{}

	benchQuery(b, buf, flushCaches,
		"eth1", "time",
		[]query.Option{
			query.WithFirst("0"),
			query.WithNumResults(query.MaxResults),
			query.WithFormat(types.FormatJSON),
			query.WithCondition("dport eq 443"),
		}...,
	)

	_ = buf
}

func BenchmarkStdQueryTableOutput(b *testing.B) {

	buf := &bytes.Buffer{}

	benchQuery(b, buf, flushCaches,
		"eth1", "time",
		[]query.Option{
			query.WithFirst("0"),
			query.WithNumResults(query.MaxResults),
		}...,
	)

	_ = buf
}

func BenchmarkStdQueryTableOutputCondition(b *testing.B) {

	buf := &bytes.Buffer{}

	benchQuery(b, buf, flushCaches,
		"eth1", "time",
		[]query.Option{
			query.WithFirst("0"),
			query.WithNumResults(query.MaxResults),
			query.WithCondition("dport eq 443"),
		}...,
	)

	_ = buf
}

func BenchmarkStdQueryCSVOutput(b *testing.B) {

	buf := &bytes.Buffer{}

	benchQuery(b, buf, flushCaches,
		"eth1", "time",
		[]query.Option{
			query.WithFirst("0"),
			query.WithNumResults(query.MaxResults),
			query.WithFormat(types.FormatCSV),
		}...,
	)

	_ = buf
}

func BenchmarkStdQueryCSVOutputCondition(b *testing.B) {

	buf := &bytes.Buffer{}

	benchQuery(b, buf, flushCaches,
		"eth1", "time",
		[]query.Option{
			query.WithFirst("0"),
			query.WithNumResults(query.MaxResults),
			query.WithCondition("dport eq 443"),
			query.WithFormat(types.FormatCSV),
		}...,
	)

	_ = buf
}

func benchQuery(b *testing.B, buf *bytes.Buffer, flushFunc func(), iface, queryStr string,
	opts ...query.Option) {
	for n := 0; n < b.N; n++ {

		// prepare query
		args := query.NewArgs(queryStr, iface, opts...).AddOutputs(buf)
		// run query
		_, err := NewQueryRunner(TestDB).Run(context.Background(), args)
		if err != nil {
			b.Fatalf("error during execute: ` + "%s" + `", err)
		}

		buf.Reset()

		if flushFunc != nil {
			flushFunc()
		}
	}
}

func flushCaches() {
	var log = logging.Logger()

	// call arch specific implementation
	cmd := exec.Command(syncCmd[0], syncCmd[1:]...)
	err := cmd.Start()
	if err != nil {
		log.Error(err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Error(err)
	}
}
`
	benchmarkTemplate = `// AUTO-GENERATED COMBINATIONS
// These benchmarks provide most input combinations to the query engines. They are
// meant for assessing the performance of specific DB accesses (e.g. long queries vs.
// short, convoluted conditions vs. none, etc.)
//
// Each benchmark follows a naming convention that allows easy selection of a subset when
// invoking "go run":
//      Benchmark + "IDENT" + q + "QUERYTYPE" + i + "IFACENAME" + n + "NUMRESULTS"
//                + o + "OUTPUTFORMAT" + c + "CONDITIONSIZE"
//
// Parameters are written ALL CAPS. IDENT can be one of the following:
//  - STD: runs bencharks with file system cache flush enabled (to test the whole pipeline)
//  - NF: doesn't flush the file system cache after every run
//
// Example: select all benchmarks running queries on eth1
//      go test -v -bench -run=BenchmarkNFqSIPDIPiETH1

// Benchmarks (AUTO-GENERATED, DO NOT EDIT)

{{ range . -}}
{{$trimmedQuery := replace .Query "," "" -1 -}}
{{$trimmedIfaces := replace .Iface "," "" -1 -}}
// Benchmark: {{ .ID }}, Query: {{.Query}}, Iface: {{.Iface}}, Condition: {{.Condition}}, N: {{.N}}
func BenchmarkNFq{{$trimmedQuery | upper}}i{{$trimmedIfaces | upper }}n{{if le .N 1000}}{{.N}}{{else}}BIG{{end}}o{{upper .Format}}c{{if ne .Condition ""}}{{upper .Condition}}{{else}}NONE{{end}}(b *testing.B) {

	buf := &bytes.Buffer{}

	benchQuery(b, buf, nil,
		"{{ .Iface }}", "{{.Query}}",
		[]query.Option{
			query.WithFirst("0"),
			query.WithNumResults({{ .N }}),
            {{if .Condition -}}
            query.WithCondition("{{ getcondition .Condition }}"),
            {{end -}}
		}...,
	)

	_ = buf
}
{{end -}}
`
)

// First we create a FuncMap with which to register the function.
var funcMap = template.FuncMap{
	"upper":        strings.ToUpper,
	"getcondition": getCondition,
	"replace":      strings.Replace,
}

// TestTuple stores a possible benchmark query configuration. It is used to enumerate different query scenarios.
type TestTuple struct {
	ID        int
	Iface     string
	Query     string
	N         uint64
	Format    string
	Condition string
}

func main() {

	var (
		masterTmpl        *template.Template
		allBenchmarksFile *os.File
		err               error
	)

	outfile := "benchmarks_test.go"

	err = os.RemoveAll(outfile)
	if err != nil {
		log.Fatalf("failed to remove previous benchmarks: %s", err)
	}

	allBenchmarksFile, err = os.OpenFile(outfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalf("failed to load static benchmarks: %s", err)
	}

	// write benchmarks base
	_, err = fmt.Fprint(allBenchmarksFile, benchmarkBase)
	if err != nil {
		log.Fatalf("failed to write static benchmarks: %s", err)
	}

	masterTmpl, err = template.New("master").Funcs(funcMap).Parse(benchmarkTemplate)
	if err != nil {
		log.Fatalf("failed to create template: %s", err)
	}

	// create the struct slice
	var (
		qarg string
		iarg string
	)

	// iterate over all queries
	benchNum := 1
	var tuples []TestTuple
	for i := range queries {
		qarg = strings.Join(queries[:i+1], ",")

		// iterate over all ifaces
		for j := range ifaces {
			iarg = strings.Join(ifaces[:j+1], ",")

			// iterate over all num results
			for _, n := range numResults {

				// iterate over all conditions
				for cname := range testConditions {

					// iterate over all formats
					for _, format := range query.PermittedFormats() {
						tuples = append(tuples, TestTuple{
							ID:        benchNum,
							Iface:     iarg,
							Query:     qarg,
							Condition: cname,
							N:         n,
							Format:    format,
						})
						benchNum++
					}
					benchNum++
				}
				benchNum++
			}
			benchNum++
		}
		benchNum++
	}

	// write combinations to file
	err = masterTmpl.Execute(allBenchmarksFile, tuples)
	if err != nil {
		log.Fatal(err)
	}

	if err = allBenchmarksFile.Close(); err != nil {
		log.Fatal(err)
	}
}

var testConditions = map[string]string{
	"none":   "", // needed to run queries without conditions
	"single": "dport eq 443",
	"double": "dport eq 443 and proto eq tcp",
	"nested": "((dport eq 443 || dport eq 80) and dport neq 8080) and ! (dnet eq 127.0.0.0/8 or dnet eq 10.0.0.0/8 or dnet eq 172.16.0.0/12 or dnet eq 192.168.0.0/16)",
}

// built-up successively (one iface, two ifaces, etc.)
var ifaces = [...]string{
	"eth0",
	"eth1",
	"eth2",
	"t_c1_fwde",
	"t_c1_fwde1",
	"tun_3g_c1_fw1",
	"tun_3g_c1_fwde",
}

// built-up successively (one attribute, two attributes, etc.)
var queries = [...]string{
	"sip",
	"dip",
	"dport",
	"proto",
	"time",
	"iface",
}

var numResults = [...]uint64{
	1,
	10,
	100,
	query.DefaultNumResults,
	query.MaxResults,
}

func getCondition(id string) string {
	return testConditions[id]
}
