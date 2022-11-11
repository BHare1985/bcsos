package dtype

type NodeInfo struct {
	Mode string `json:"mode"`
	SC   int    `json:"storage_class"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Hash string `json:"hash"`
}

type ReqData struct {
	Addr      string `json:"Addr"`
	Timestamp int64  `json:"Timestamp"`
	SC        int    `json:"storage_class"`
	Hop       int    `json:"Hop"`
	ObjType   string `json:"ObjType"`
	ObjHash   string `json:"ObjHash"`
}

type Command struct {
	Cmd    string `json:"cmd"`
	Subcmd string `json:"subcmd"`
	Arg1   string `json:"arg1"`
	Arg2   string `json:"arg2"`
	Arg3   string `json:"arg3"`
}

type ReqConsecutiveHashes struct {
	Hash  string `json:"Hash"`
	Count int    `json:"Count"`
}

type ResConsecutiveHashes struct {
	Hashes string `json:"Hashes"`
}

type ReqEncryptedBlock struct {
	Hash string `json:"Hash"`
}

type ResEncryptedBlock struct {
	Block string `json:"Block"`
}

type PoSProof struct {
	Timestamp int64
	Address   []byte
	Root      []byte
	HashEncs  [][]byte
	HashKeys  [][]byte
	Selected  int
	Hash      string
	EncBlock  []byte
}
