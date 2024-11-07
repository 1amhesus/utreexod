package main

import (
	"compress/bzip2"
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

	return blocks, nil
}

// generateRoots processes each block and calculates the Utreexo root.
func generateRoots(blocks []*btcutil.Block) (map[int][]byte, error) {
	accumulator := utreexo.NewAccumulator()
	rootMap := make(map[int][]byte)

	for i, block := range blocks {
		blockHeight := i // Block height starts at 0

		leaves := []utreexo.Leaf{{Hash: utreexo.Hash(block.Hash().CloneBytes())}}
		deletes := []utreexo.Hash{}
		proofs := utreexo.Proof{}

		err := accumulator.Modify(leaves, deletes, proofs)
		if err != nil {
			return nil, fmt.Errorf("failed to modify Utreexo accumulator for block %d: %v", blockHeight, err)
		}

		rootMap[blockHeight] = accumulator.GetRoots()[0][:]
	}

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

	for _, height := range blockHeights {
		fmt.Printf("Block Height %d: Utreexo Root %x\n", height, roots[height])
	}
}