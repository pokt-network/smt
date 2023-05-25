package delete

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/pokt-network/smt"
)

func Fuzz(data []byte) int {
	if len(data) == 0 {
		return -1
	}

	splits := bytes.Split(data, []byte("*"))
	if len(splits) < 3 {
		return -1
	}

	smn := smt.NewSimpleMap()
	tree := smt.NewSparseMerkleTree(smn, sha256.New())
	for i := 0; i < len(splits)-1; i += 2 {
		key, value := splits[i], splits[i+1]
		if err := tree.Update(key, value); err != nil {
			panic(fmt.Sprintf("error updating key: %s (%s)", key, err.Error()))
		}
	}

	deleteKey := splits[len(splits)-1]
	if err := tree.Delete(deleteKey); err != nil {
		panic(fmt.Sprintf("error deleting key: %s (%s)", deleteKey, err.Error()))
	}

	newRoot := tree.Root()

	if len(newRoot) == 0 {
		panic("newRoot is nil yet err==nil")
	}

	return 1
}
