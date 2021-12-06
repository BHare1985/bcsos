package storage

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/junwookheo/bcsos/common/blockchain"
	"github.com/junwookheo/bcsos/common/config"
	"github.com/junwookheo/bcsos/common/dbagent"
	"github.com/junwookheo/bcsos/common/dtype"
	"github.com/junwookheo/bcsos/storagesrv/network"
	"github.com/junwookheo/bcsos/storagesrv/testmgrcli"
)

type Handler struct {
	http.Handler
	db    dbagent.DBAgent
	sim   dtype.NodeInfo
	local dtype.NodeInfo
	tmc   *testmgrcli.TestMgrCli
	nm    *network.NodeMgr
	om    *ObjectMgr
	mutex sync.Mutex
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *Handler) Stop() {
	h.db.Close()
}

// newBlockHandler is called when a new block is received from miners
// When a node receive this, it stores the block on its local db
// Request : a new block
// Response : none
func (h *Handler) newBlockHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("newBlockHandler", err)
		return
	}
	defer ws.Close()

	var block blockchain.Block
	if err := ws.ReadJSON(&block); err != nil {
		log.Printf("Read json error : %v", err)
	}

	h.db.AddBlock(&block)
	// ws.WriteJSON(block)
	// for _, t := range block.Transactions {
	// 	log.Printf("From client : %s", t.Data)
	// }
}

// getTransactionHandler is called when transaction query from other nodes is received
// if the node does not have the transaction, the node will query it to other nodes with highr SC
// Request : hash of transaction
// Response : transaction
func (h *Handler) getObjectHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("getTransactionHandler transaction error : ", err)
		return
	}
	defer ws.Close()

	defer h.db.UpdateDBNetworkQuery(1, 0, 1)
	var reqData dtype.ReqData
	if err := ws.ReadJSON(&reqData); err != nil {
		log.Printf("Read json error : %v", err)
		return
	}

	var obj interface{}
	if reqData.ObjType == "transaction" {
		tr := blockchain.Transaction{}
		if h.db.GetTransaction(reqData.ObjHash, &tr) == 0 {
			if h.getObjectQuery(h.local.SC+1, &reqData, &tr) {
				h.db.AddTransaction(&tr)
			}
		}
		obj = tr
	} else if reqData.ObjType == "blockheader" {
		bh := blockchain.BlockHeader{}
		if h.db.GetBlockHeader(reqData.ObjHash, &bh) == 0 {
			if h.getObjectQuery(h.local.SC+1, &reqData, &bh) {
				h.db.AddBlockHeader(reqData.ObjHash, &bh)
			}
		}
		obj = bh
	} else {
		log.Panicf("Not support object type")
	}

	reqData.Addr = fmt.Sprintf("%v:%v", h.local.IP, h.local.Port)
	reqData.Hop += 1

	ws.WriteJSON(reqData)
	ws.WriteJSON(obj)
	log.Printf("<==Query write reqData: %v", reqData)
}

func (h *Handler) newReqData(objtype string, hash string) dtype.ReqData {
	req := dtype.ReqData{}
	req.Addr = fmt.Sprintf("%v:%v", h.local.IP, h.local.Port)
	req.Timestamp = time.Now().UnixNano()
	req.Hop = 0
	req.ObjType = objtype
	req.ObjHash = hash

	return req
}

// getTransactionQuery queries a transaction ot other nodes with highr Storage Class
// Request : hash of transaction
// Response : transaction
func (h *Handler) getObjectQuery(startSC int, reqData *dtype.ReqData, obj interface{}) bool {
	queryObject := func(ip string, port int, reqData *dtype.ReqData, obj interface{}) bool {
		url := fmt.Sprintf("ws://%v:%v/getobject", ip, port)
		//log.Printf("getTransactionQuery : %v", url)

		ws, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Printf("getTransactionQuery Dial error : %v", err)
			return false
		}
		defer ws.Close()

		// the number of query to other nodes
		defer h.db.UpdateDBNetworkQuery(0, 1, 1)

		if err := ws.WriteJSON(*reqData); err != nil {
			log.Printf("Write json error : %v", err)
			return false
		}

		if err := ws.ReadJSON(reqData); err != nil {
			log.Printf("Read json error : %v", err)
			return false
		}
		if err := ws.ReadJSON(obj); err != nil {
			log.Printf("Read json error : %v", err)
			return false
		}

		h.db.UpdateDBNetworkDelay(int(time.Now().UnixNano()-reqData.Timestamp), reqData.Hop)
		log.Printf("==>Query read reqData: %v", reqData)
		return true
	}

	for i := startSC; i < config.MAX_SC; i++ {
		var nodes [config.MAX_SC_PEER]dtype.NodeInfo
		if h.nm.GetSCNNodeListbyDistance(i, reqData.ObjHash, &nodes) {
			for _, node := range nodes {
				if node.IP == "" {
					continue
				}
				if node.Hash == h.local.Hash { // If the node is itself, skip
					continue
				}

				if queryObject(node.IP, node.Port, reqData, obj) {
					return true
				}
				//time.Sleep(time.Duration(200 * time.Microsecond.Seconds()))
				log.Printf("queryObject fail : query other nodes")
			}
		}
	}

	return false
}

// Response to web app with dbstatus information
// keep sending dbstatus to the web app
func (h *Handler) nodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("newBlockHandler", err)
		return
	}
	defer ws.Close()
	h.tmc.NodeInfoHandler(ws, w, r)
}

// Send response to connector with its local information
func (h *Handler) pingHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("pingHandler", err)
		return
	}
	defer ws.Close()

	peer := dtype.NodeInfo{}
	if err := ws.ReadJSON(&peer); err != nil {
		log.Printf("Read json error : %v", err)
		return
	}
	//log.Printf("receive peer addr : %v", peer)

	// Received peer info so add it to peer list
	if peer.Hash != "" && peer.Hash != h.local.Hash {
		h.nm.AddNSCNNode(peer)
	}

	// Send peers info to the connector
	var nodes [(config.MAX_SC) * config.MAX_SC_PEER]dtype.NodeInfo
	h.nm.GetSCNNodeListAll(&nodes)
	if err := ws.WriteJSON(nodes); err != nil {
		log.Printf("Write json error : %v", err)
		return
	}
}

// Response to web app with dbstatus information
// keep sending dbstatus to the web app
func (h *Handler) endTestHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("endTestHandler", err)
		return
	}
	defer ws.Close()
	var endtest string
	if err := ws.ReadJSON(&endtest); err != nil {
		log.Printf("Read json error : %v", err)
		return
	}

	if endtest == config.END_TEST {
		log.Println("Received End test")
		h.db.Close()
		time.Sleep(3 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}
}

func (h *Handler) ObjectbyAccessPatternProc() {
	go func() {
		ticker := time.NewTicker(time.Duration(config.TIME_AP_GEN) * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			hashes := []dbagent.RemoverbleObj{}
			ret := false

			if config.ACCESS_FREQUENCY_PATTERN == config.RANDOM_ACCESS_PATTERN {
				ret = h.om.AccessWithUniform(config.NUM_AP_GEN, &hashes)
			} else {
				ret = h.om.AccessWithExponential(config.NUM_AP_GEN, &hashes)
			}

			if ret {
				for _, hash := range hashes {
					if hash.HashType == 0 {
						bh := blockchain.BlockHeader{}
						req := h.newReqData("blockheader", hash.Hash)
						if h.getObjectQuery(h.local.SC, &req, &bh) {
							h.db.AddBlockHeader(hash.Hash, &bh)
							if hash.Hash != hex.EncodeToString(bh.GetHash()) {
								log.Panicf("%v header Hash not equal %v", hash.Hash, hex.EncodeToString(bh.GetHash()))
							}
						}
					} else {
						tr := blockchain.Transaction{}
						req := h.newReqData("transaction", hash.Hash)
						if h.getObjectQuery(h.local.SC, &req, &tr) {
							h.db.AddTransaction(&tr)
							if hash.Hash != hex.EncodeToString(tr.Hash) {
								log.Panicf("%v Tr Hash not equal %v", hash.Hash, hex.EncodeToString(tr.Hash))
							}
						}
					}
				}

				if h.local.SC < config.MAX_SC-1 {
					h.om.DeleteNoAccedObjects()
				}
			}

			status := h.om.db.GetDBStatus()
			log.Printf("Status : %v", status)
		}
	}()
}

func (h *Handler) PeerListProc() {
	go func() {
		ticker := time.NewTicker(time.Duration(config.TIME_UPDATE_NEITHBOUR) * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			if h.sim.IP != "" && h.sim.Port != 0 && h.local.Hash != "" {
				h.nm.UpdatePeerList(h.sim, h.local)
			}
		}
	}()
}

func NewHandler(path string, local dtype.NodeInfo) *Handler {
	m := mux.NewRouter()
	h := &Handler{
		Handler: m,
		db:      dbagent.NewDBAgent(path, local.SC),
		sim:     dtype.NodeInfo{Mode: "", SC: config.SIM_SC, IP: "", Port: 0, Hash: ""},
		local:   local,
		tmc:     nil,
		nm:      nil,
		om:      nil,
		mutex:   sync.Mutex{},
	}

	h.tmc = testmgrcli.NewTMC(h.db, &h.sim, &h.local)
	h.nm = network.NewNodeMgr(&h.local)
	h.om = NewObjMgr(h.db)

	m.Handle("/", http.FileServer(http.Dir("static")))
	m.HandleFunc("/newblock", h.newBlockHandler)
	m.HandleFunc("/getobject", h.getObjectHandler)
	m.HandleFunc("/nodeinfo", h.nodeInfoHandler)
	m.HandleFunc("/ping", h.pingHandler)
	m.HandleFunc("/endtest", h.endTestHandler)
	return h
}
