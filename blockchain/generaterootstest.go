package main

import (
	"compress/bzip2"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/utreexo/utreexo"
	"github.com/utreexo/utreexod/btcutil"
)

// loadBlocks reads binary block data from a compressed file in the testdata directory.
func loadBlocks(filename string) ([]*btcutil.Block, error) {
	filePath := filepath.Join("testdata", filename)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	var blocks []*btcutil.Block
	reader := bzip2.NewReader(file)

	for {
		var magic [4]byte
		err := binary.Read(reader, binary.LittleEndian, &magic)
		if err == io.EOF {
			break // End of file reached
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read block magic: %v", err)
		}

		var blockLen uint32
		err = binary.Read(reader, binary.LittleEndian, &blockLen)
		if err != nil {
			return nil, fmt.Errorf("failed to read block length: %v", err)
		}

		blockData := make([]byte, blockLen)
		_, err = io.ReadFull(reader, blockData)
		if err != nil {
			return nil, fmt.Errorf("failed to read block data: %v", err)
		}

		block, err := btcutil.NewBlockFromBytes(blockData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse block data: %v", err)
		}

		blocks = append(blocks, block)
	}

	for i, block := range blocks {
		fmt.Printf("Block %d: Hash: %s, Transactions: %d\n", i, block.Hash(), len(block.Transactions()))
	}
	
	return blocks, nil
}
// generateRoots processes each block and calculates the Utreexo root.
func generateRoots(blocks []*btcutil.Block) (map[int][][]byte, error) {
	accumulator := utreexo.NewAccumulator()
	rootMap := make(map[int][][]byte)

	for i, block := range blocks {
		blockHeight := i
		var leaves []utreexo.Leaf
		var deletes []utreexo.Hash
	
		for _, tx := range block.Transactions() {
			txHash := tx.Hash()
			for _, txOut := range tx.MsgTx().TxOut {
				//hash := sha256.Sum256(txOut.PkScript)
				//leaf := utreexo.Leaf{Hash: utreexo.Hash(hash[:])}
				combinedData := append(txHash.CloneBytes(), txOut.PkScript...)
        		outHash := sha256.Sum256(combinedData)

				leaf := utreexo.Leaf{Hash: utreexo.Hash(outHash[:])}
				leaves = append(leaves, leaf)
				//fmt.Printf("Block %d - Adding leaf for TxOut: %x\n", blockHeight, outHash[:])
			}
	
			seenDeletes := make(map[string]bool)
			for _, txIn := range tx.MsgTx().TxIn {
				deleteHash := utreexo.Hash(txIn.PreviousOutPoint.Hash.CloneBytes())
				if !seenDeletes[string(deleteHash[:])] {
					deletes = append(deletes, deleteHash)
					seenDeletes[string(deleteHash[:])] = true
					//fmt.Printf("Block %d - Adding delete for TxIn: %x\n", blockHeight, deleteHash[:])
				}
			}
		}

		// Modify 전 루트 상태 확인
		fmt.Printf("Roots before Modify for Block %d: %v\n", blockHeight, accumulator.GetRoots())

		// leaves와 deletes 내용 확인
		//fmt.Printf("Block %d - Leaves Count: %d, Deletes Count: %d\n", blockHeight, len(leaves), len(deletes))
	
		proofs := utreexo.Proof{}
		err := accumulator.Modify(leaves, deletes, proofs)
		if err != nil {
			return nil, fmt.Errorf("failed to modify Utreexo accumulator for block %d: %v", blockHeight, err)
		}
	
		/**
		roots := accumulator.GetRoots()
		if len(roots) > 0 {
			rootMap[blockHeight] = roots[0][:]
			fmt.Printf("Block Height %d: Utreexo Root %x\n", blockHeight, roots[0][:])
		} else {
			fmt.Printf("Block Height %d: No roots generated\n", blockHeight)   
		}
		**/
		// Modify 후 루트 상태 확인
		roots := accumulator.GetRoots()
		rootMap[blockHeight] = make([][]byte, len(roots))
		for j, root := range roots {
			rootMap[blockHeight][j] = root[:]
			fmt.Printf("Block Height %d: Utreexo Root %x\n", blockHeight, root[:])
		}

		fmt.Printf("Accumulator Roots after Block %d: %v\n", blockHeight, accumulator.GetRoots())
		
	}
	/**
	for i, block := range blocks {
		fmt.Printf("Block Height %d - Transactions Count: %d\n", i, len(block.Transactions()))
	
		for j, tx := range block.Transactions() {
			fmt.Printf("  Tx %d - Hash: %s\n", j, tx.Hash())
		}
	}
	**/
	return rootMap, nil
}

func main() {
	blocks, err := loadBlocks("blk_0_to_4.dat.bz2")
	if err != nil {
		log.Fatalf("Error loading blocks: %v", err)
	}

	roots, err := generateRoots(blocks)
	if err != nil {
		log.Fatalf("Error generating roots: %v", err)
	}

	blockHeights := make([]int, 0, len(roots))
	for height := range roots {
		blockHeights = append(blockHeights, height)
	}
	sort.Ints(blockHeights)

	/**
	for _, height := range blockHeights {
		fmt.Printf("Block Height %d: Utreexo Root %x\n", height, roots[height])
	}
	**/
}