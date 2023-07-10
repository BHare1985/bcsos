# PPoS
This is a simulator for [PPoS: Practical Proof of Storage for Blockchain Full Nodes](https://eprints.qut.edu.au/238319/) which was presented at the ICBC2023 conference.

# Requirements
* go version 1.16 
* Linux 64bit/Raspberry 64bit/Windows 10 or above 64bit
* tmux 3.0a

# Run
* cd blockchainnode
* run simxx.sh
* connect http://localhost:8080/ and click "Start Test" button

## Runing individual nodes
* cd blockchainnode
* go run blockchainnode.go -mode=ST -sc=0 -port=7001'
  * sc is the storage class, 3 is the highest node
  * mode : ST(Server will generate transactions and access objects) or MI(A node will generate transactions and access objects)

## Runing Simulator server
* cd blockchainsim
* go run blockchainsim.go




