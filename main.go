package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
)

const hashLeadingZerosCount = 1

var blockChain []Block
var blockChainMutex sync.Mutex

type Block struct {
	timeStamp    time.Time
	data         string
	nonce        string
	hash         string
	previousHash string
}

func main() {

	var block0 Block

	block0.timeStamp = time.Now()
	block0.data = "First block test"

	hash, err := generateHash(block0)
	if err != nil {
		log.Fatal(err)
	}

	block0.hash = hash
	blockChain = append(blockChain, block0)

	router := mux.NewRouter()

	router.HandleFunc("/", getBlockChainHandler).Methods("GET")
	router.HandleFunc("/write", postBlockHandler).Methods("POST")

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}

func getBlockChainHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, spew.Sdump(blockChain))
}

func postBlockHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respond(w, http.StatusBadRequest, "Could not read body")
		return
	}

	prevBlock := blockChain[len(blockChain)-1]
	block, err := createBlock(prevBlock.hash, string(body))
	if err != nil {
		respond(w, http.StatusBadRequest, "Could not create block")
		return
	}

	blockChainMutex.Lock()

	if !isBlockHashValid(block, blockChain[len(blockChain)-1]) {
		respond(w, http.StatusInternalServerError, "Block hash is not valid")
		return
	}

	blockChain = append(blockChain, block)
	blockChainMutex.Unlock()

	respond(w, http.StatusCreated, "success")
}

func respond(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)

	_, err := w.Write([]byte(message))
	if err != nil {
		fmt.Printf("Could not send response, %s", err)
	}

}

func isBlockHashValid(block, oldBlock Block) bool {
	if block.previousHash != oldBlock.hash {
		return false
	}

	hash, err := generateHash(block)
	if err != nil {
		fmt.Printf("Could not generate Hash for block validation, %s", err)
		return false
	}

	if hash != block.hash {
		return false
	}

	return true
}

func createBlock(prevBlockHash string, data string) (Block, error) {
	var block Block

	block.timeStamp = time.Now()
	block.data = data
	block.previousHash = prevBlockHash

	for i := 0; ; i++ {

		block.nonce = fmt.Sprintf("%x", i)

		hash, err := generateHash(block)
		if err != nil {
			fmt.Printf("Could not add Hash to Block, %s", err)
			return block, err
		}

		if !hasZerosAsPrefix(hash) {
			fmt.Println("Looking for block")
			continue
		}

		fmt.Println("Block was found")
		block.hash = hash
		break
	}

	return block, nil
}

func generateHash(block Block) (string, error) {

	dataToHash := block.timeStamp.String() + block.data + block.previousHash + block.nonce

	hash := sha256.New()
	_, err := hash.Write([]byte(dataToHash))
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(hash.Sum(nil)), nil
}

func hasZerosAsPrefix(hash string) bool {
	prefix := strings.Repeat("0", hashLeadingZerosCount)
	return strings.HasPrefix(hash, prefix)
}
