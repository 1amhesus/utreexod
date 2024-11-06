// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2018-2021 The Decred developers
// Copyright (c) 2024 The Utreexod developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
// TestBlockExists tests the blockExists function in a non-database environment.

package blockchain_test

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/utreexo/utreexod/blockchain"
	"github.com/utreexo/utreexod/chaincfg"
	"github.com/utreexo/utreexod/wire"
)

// TestProcessBlockHeader tests the ProcessBlockHeader function using pre-loaded block headers.
func TestProcessBlockHeader(t *testing.T) {
	// Load up the first few blocks for header testing.
	// (genesis block) -> 1 -> 2 -> 3 -> 4
	testFiles := []string{
		"blk_0_to_4.dat.bz2",
	}

	var headers []*wire.BlockHeader
	for _, file := range testFiles {
		fileHeaders, err := loadBlockHeaders(file)
		if err != nil {
			t.Errorf("Error loading file: %v\n", err)
			return
		}
		headers = append(headers, fileHeaders...)
	}

	// Initialize a blockchain instance without a database.
	chain, err := blockchain.New(&blockchain.Config{
		ChainParams: &chaincfg.MainNetParams,
		TimeSource:  blockchain.NewMedianTime(),
	})
	require.NoError(t, err, "Failed to create blockchain instance")

	// Process each block header and ensure it's accepted without errors.
	for i, header := range headers {
		err := chain.ProcessBlockHeader(header)
		if err != nil {
			t.Errorf("ProcessBlockHeader failed on header %d: %v\n", i, err)
		}
	}
}

// loadBlockHeaders reads block headers from a test data file.
func loadBlockHeaders(filename string) ([]*wire.BlockHeader, error) {
	// Set up the path to the test data file.
	testDataFile := filepath.Join("testdata", filename)
	file, err := os.Open(testDataFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bzip2.NewReader(file) // Decompress .bz2 file

	var headers []*wire.BlockHeader

	// Only read a few headers to avoid high memory usage.
	const maxHeaders = 5
	headerCount := 0

	for headerCount < maxHeaders {
		// Each block starts with an 8-byte length prefix.
		var blockLengthBytes [8]byte
		_, err := reader.Read(blockLengthBytes[:])
		if err != nil {
			break // Assume EOF or valid end of file
		}
		blockLength := binary.LittleEndian.Uint64(blockLengthBytes[:])

		// Read the full block data based on the length prefix.
		blockData := make([]byte, blockLength)
		_, err = reader.Read(blockData)
		if err != nil {
			return nil, err
		}

		// Deserialize the block to extract its header.
		var msgBlock wire.MsgBlock
		err = msgBlock.Deserialize(bytes.NewReader(blockData))
		if err != nil {
			return nil, err
		}

		// Append the header to the headers slice.
		headers = append(headers, &msgBlock.Header)
		headerCount++
	}

	return headers, nil
}



