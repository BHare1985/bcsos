package dtype

type NodeInfo struct {
	Mode string `json:"mode"`
	Type string `json:"type"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Hash string `json:"hash"`
}

type Simulator struct {
	IP   string
	Port int
}

type Version struct {
	Ver int
}
