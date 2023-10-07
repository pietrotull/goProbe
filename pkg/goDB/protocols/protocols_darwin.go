//go:build darwin
// +build darwin

// Code generated by protocols_generator.go - DO NOT EDIT.
package protocols

// IPProtocols stores the IP protocol mappings to friendly name
var IPProtocols = map[int]string{
	0:   "IP",
	1:   "ICMP",
	2:   "IGMP",
	3:   "GGP",
	4:   "IP-ENCAP",
	5:   "ST2",
	6:   "TCP",
	7:   "CBT",
	8:   "EGP",
	9:   "IGP",
	10:  "BBN-RCC-MON",
	11:  "NVP-II",
	12:  "PUP",
	13:  "ARGUS",
	14:  "EMCON",
	15:  "XNET",
	16:  "CHAOS",
	17:  "UDP",
	18:  "MUX",
	19:  "DCN-MEAS",
	20:  "HMP",
	21:  "PRM",
	22:  "XNS-IDP",
	23:  "TRUNK-1",
	24:  "TRUNK-2",
	25:  "LEAF-1",
	26:  "LEAF-2",
	27:  "RDP",
	28:  "IRTP",
	29:  "ISO-TP4",
	30:  "NETBLT",
	31:  "MFE-NSP",
	32:  "MERIT-INP",
	33:  "DCCP",
	34:  "3PC",
	35:  "IDPR",
	36:  "XTP",
	37:  "DDP",
	38:  "IDPR-CMTP",
	39:  "TP++",
	40:  "IL",
	41:  "IPV6",
	42:  "SDRP",
	43:  "IPV6-ROUTE",
	44:  "IPV6-FRAG",
	45:  "IDRP",
	46:  "RSVP",
	47:  "GRE",
	48:  "DSR",
	49:  "BNA",
	50:  "ESP",
	51:  "AH",
	52:  "I-NLSP",
	53:  "SWIPE",
	54:  "NARP",
	55:  "MOBILE",
	56:  "TLSP",
	57:  "SKIP",
	58:  "IPV6-ICMP",
	59:  "IPV6-NONXT",
	60:  "IPV6-OPTS",
	62:  "CFTP",
	64:  "SAT-EXPAK",
	65:  "KRYPTOLAN",
	66:  "RVD",
	67:  "IPPC",
	69:  "SAT-MON",
	70:  "VISA",
	71:  "IPCV",
	72:  "CPNX",
	73:  "CPHB",
	74:  "WSN",
	75:  "PVP",
	76:  "BR-SAT-MON",
	77:  "SUN-ND",
	78:  "WB-MON",
	79:  "WB-EXPAK",
	80:  "ISO-IP",
	81:  "VMTP",
	82:  "SECURE-VMTP",
	83:  "VINES",
	84:  "TTP",
	85:  "NSFNET-IGP",
	86:  "DGP",
	87:  "TCF",
	88:  "EIGRP",
	89:  "OSPFIGP",
	90:  "Sprite-RPC",
	91:  "LARP",
	92:  "MTP",
	93:  "AX.25",
	94:  "IPIP",
	95:  "MICP",
	96:  "SCC-SP",
	97:  "ETHERIP",
	98:  "ENCAP",
	100: "GMTP",
	101: "IFMP",
	102: "PNNI",
	103: "PIM",
	104: "ARIS",
	105: "SCPS",
	106: "QNX",
	107: "A/N",
	108: "IPComp",
	109: "SNP",
	110: "Compaq-Peer",
	111: "IPX-in-IP",
	112: "CARP",
	113: "PGM",
	115: "L2TP",
	116: "DDX",
	117: "IATP",
	118: "STP",
	119: "SRP",
	120: "UTI",
	121: "SMP",
	122: "SM",
	123: "PTP",
	124: "ISIS",
	125: "FIRE",
	126: "CRTP",
	127: "CRUDP",
	128: "SSCOPMCE",
	129: "IPLT",
	130: "SPS",
	131: "PIPE",
	132: "SCTP",
	133: "FC",
	134: "RSVP-E2E-IGNORE",
	135: "Mobility-Header",
	136: "UDPLite",
	137: "MPLS-IN-IP",
	138: "MANET",
	139: "HIP",
	140: "SHIM6",
	141: "WESP",
	142: "ROHC",
	240: "PFSYNC",
	255: "UNKNOWN",
}

// IPProtocolIDs is the reverse mapping from friendly name to protocol number
var IPProtocolIDs = map[string]int{
	"3pc":             34,
	"a/n":             107,
	"ah":              51,
	"argus":           13,
	"aris":            104,
	"ax.25":           93,
	"bbn-rcc-mon":     10,
	"bna":             49,
	"br-sat-mon":      76,
	"carp":            112,
	"cbt":             7,
	"cftp":            62,
	"chaos":           16,
	"cphb":            73,
	"cpnx":            72,
	"crtp":            126,
	"crudp":           127,
	"compaq-peer":     110,
	"dccp":            33,
	"dcn-meas":        19,
	"ddp":             37,
	"ddx":             116,
	"dgp":             86,
	"dsr":             48,
	"egp":             8,
	"eigrp":           88,
	"emcon":           14,
	"encap":           98,
	"esp":             50,
	"etherip":         97,
	"fc":              133,
	"fire":            125,
	"ggp":             3,
	"gmtp":            100,
	"gre":             47,
	"hip":             139,
	"hmp":             20,
	"i-nlsp":          52,
	"iatp":            117,
	"icmp":            1,
	"idpr":            35,
	"idpr-cmtp":       38,
	"idrp":            45,
	"ifmp":            101,
	"igmp":            2,
	"igp":             9,
	"il":              40,
	"ip":              0,
	"ip-encap":        4,
	"ipcv":            71,
	"ipcomp":          108,
	"ipip":            94,
	"iplt":            129,
	"ippc":            67,
	"ipv6":            41,
	"ipv6-frag":       44,
	"ipv6-icmp":       58,
	"ipv6-nonxt":      59,
	"ipv6-opts":       60,
	"ipv6-route":      43,
	"ipx-in-ip":       111,
	"irtp":            28,
	"isis":            124,
	"iso-ip":          80,
	"iso-tp4":         29,
	"kryptolan":       65,
	"l2tp":            115,
	"larp":            91,
	"leaf-1":          25,
	"leaf-2":          26,
	"manet":           138,
	"merit-inp":       32,
	"mfe-nsp":         31,
	"micp":            95,
	"mobile":          55,
	"mpls-in-ip":      137,
	"mtp":             92,
	"mux":             18,
	"mobility-header": 135,
	"narp":            54,
	"netblt":          30,
	"nsfnet-igp":      85,
	"nvp-ii":          11,
	"ospfigp":         89,
	"pfsync":          240,
	"pgm":             113,
	"pim":             103,
	"pipe":            131,
	"pnni":            102,
	"prm":             21,
	"ptp":             123,
	"pup":             12,
	"pvp":             75,
	"qnx":             106,
	"rdp":             27,
	"rohc":            142,
	"rsvp":            46,
	"rsvp-e2e-ignore": 134,
	"rvd":             66,
	"sat-expak":       64,
	"sat-mon":         69,
	"scc-sp":          96,
	"scps":            105,
	"sctp":            132,
	"sdrp":            42,
	"secure-vmtp":     82,
	"shim6":           140,
	"skip":            57,
	"sm":              122,
	"smp":             121,
	"snp":             109,
	"sps":             130,
	"srp":             119,
	"sscopmce":        128,
	"st2":             5,
	"stp":             118,
	"sun-nd":          77,
	"swipe":           53,
	"sprite-rpc":      90,
	"tcf":             87,
	"tcp":             6,
	"tlsp":            56,
	"tp++":            39,
	"trunk-1":         23,
	"trunk-2":         24,
	"ttp":             84,
	"udp":             17,
	"udplite":         136,
	"unknown":         255,
	"uti":             120,
	"vines":           83,
	"visa":            70,
	"vmtp":            81,
	"wb-expak":        79,
	"wb-mon":          78,
	"wesp":            141,
	"wsn":             74,
	"xnet":            15,
	"xns-idp":         22,
	"xtp":             36,
}