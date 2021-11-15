package dbagent

import (
	"time"

	"github.com/junwookheo/bcsos/common/blockchain"
)

type DBAgent interface {
	Close()
	GetLatestBlockHash() string
	RemoveObject(hash string) bool
	AddBlockHeader(hash string, h *blockchain.BlockHeader) int64
	GetBlockHeader(hash string, h *blockchain.BlockHeader) int64
	AddTransaction(t *blockchain.Transaction) int64
	GetTransaction(hash string, t *blockchain.Transaction) int64
	AddBlock(b *blockchain.Block) int64
	GetBlock(hash string, b *blockchain.Block) int64
	ShowAllObjets() bool
	GetDBDataSize() uint64
	GetDBStatus() *DBStatus
	GetTransactionwithRandom(num int) *[]RemoverbleObj
	GetTransactionwithTimeWeight(num int) *[]RemoverbleObj
	DeleteNoAccedObjects()
	UpdateDBNetworkOverhead(fromqc int, toqc int)
}

type StorageObj struct {
	Type      string
	Hash      string
	Timestamp int64
	Data      interface{}
}

type StorageBLTR struct {
	Blockhash       string
	index           int
	Transactionhash string
	ACTime          int64
	AFLever         int
}
type DBStatus struct {
	ID                int
	TotalBlocks       int
	TotalTransactoins int
	Headers           int
	Blocks            int
	Transactions      int
	Size              int
	Overheadfrom      int // the number of received query
	Overheadto        int // the number of send query
	Timestamp         time.Time
}

type RemoverbleObj struct {
	HashType int // blockheader == 0 otherwise transaction
	Hash     string
}

func NewDBAgent(path string, afl int) DBAgent {
	return newDBSqlite(path, afl)
}
