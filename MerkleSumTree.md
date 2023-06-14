# Sparse Merkle Sum Tree (smst) <!-- omit in toc -->

- [Overview](#overview)
- [Implementation](#implementation)
  - [Hexadecimal Encoding](#hexadecimal-encoding)
  - [Digests](#digests)
  - [Sum Leaves](#sum-leaves)
  - [Visualisations](#visualisations)
    - [General Tree Structure](#general-tree-structure)
    - [Sum Hex Digests](#sum-hex-digests)
- [Sum](#sum)
- [Example](#example)

## Overview

Merkle Sum trees function very similarly to regular Merkle trees, with the primary difference being that each leaf node in a Merkle sum tree includes a `sum` in addition to its value. This allows for the entire tree's total sum to be calculated easily, as the sum of any branch is the sum of its children. Thus the sum of the root node is the sum of the entire tree. Like a normal Merkle tree, the Merkle sum tree allows for the efficient verification of its members, proving non-membership / membership of certain elements and generally functions the same.

Merkle sum trees can be very useful for blockchain applications in that they can easily track accounts balances and, thus, the total balance of all accounts. They can be very useful in proof of reserve systems whereby one needs to prove the membership of an element that is a component of the total sum, along with a verifiable total sum of all elements.

## Implementation

The implementation of the Sparse Merkle Sum Tree (SMST) follows in principal the same implementation as the [Plasma Core Merkle Sum tree][plasma core docs]. The main differences with the current SMT implementation are detailed below, the SMST differs mainly in the encoding of node data within the tree to accommodate for the sum.

### Hexadecimal Encoding

The sum for any node is encoded in a hexadecimal byte array with a fixed size (`[8]byte`) this allows for the sum to fully represent a `uint64` value in hexadecimal form. The golang `encoding/hex` package is used to encode the result of `fmt.Sprintf("%016x", uint64(sum))` into a byte array.

### Digests

The digest for any node in the SMST is calculated in partially the same manner as the regular SMT. The main differences are that the sum is included in the digest preimage - meaning the hash of any node's data includes its sum. But also the sum is appended to the digest of the node, resulting in a digest that looks like:

`digest = [node digest]+[8 byte hex sum]`

Therefore for the following node types the digests are as follows:

- Inner Nodes
  - Prefixed `[]byte{1}`
  - `digest = hash([]byte{1} + leftChild.digest + rightChild.digest + hex(leftChildSum+rightChildSum))+[8 byte hex sum]`
- Extension Nodes
  - Prefixed `[]byte{2}`
  - `digest = hash([]byte{2} + pathBounds + path + child.digest + hex(childSum))+[8 byte hex sum]`
- Sum Leaf Nodes
  - Prefixed `[]byte{0}`
  - `digest = hash([]byte{0} + path + value + hexSum)+[8 byte hex sum]`
- Lazy Nodes
  - Prefix of the actual node type is stored in the digest
  - `digest = persistedDigest`

This means that with a hasher such as `sha256.New()` whose hash size is 32 bytes, the digest of any node will be 40 bytes in length.

### Sum Leaves

The SMST introduces a new node type, the `sumLeafNode` which is almost identical to a `leafNode` from the SMT. However, it includes a `sum` field which is a `[8]byte` hexadecimal representation of the `uint64` sum of the node. In an SMST the `sumLeafNode` replaces the `leafNode` type.

### Visualisations

The following diagrams are representations of how the tree and its components can be visualised.

#### General Tree Structure

In reality the only nodes that actually hold a sum value are the `sumLeafNode` types and all other nodes store their sum as part of their digest. However, for the sake of visualisation the sum is included in all nodes.

```mermaid
graph TB
	subgraph Root
		A1["Digest: Hash(Hash(Path+Hash1)+Hash(Hash2+(Hash(Hash3+Hash4)))+Hex(20))+Hex(20)"]
        A2[Sum: 20]
	end
	subgraph BI[Inner Node]
		B1["Digest: Hash(Hash2+(Hash(Hash3+Hash4))+Hex(12))+Hex(12)"]
        B2[Sum: 12]
	end
	subgraph BE[Extension Node]
		B3["Digest: Hash(Path+Hash1+Hex(8))+Hex(8)"]
        B4[Sum: 8]
	end
	subgraph CI[Inner Node]
		C1["Digest: Hash(Hash3+Hash4+Hex(7))+Hex(7)"]
        C2[Sum: 7]
	end
	subgraph CL[Sum Leaf Node]
		C3[Hash2]
        C4[Sum: 5]
	end
	subgraph DL1[Sum Leaf Node]
		D1[Hash3]
        D2[Sum: 4]
	end
	subgraph DL2[Sum Leaf Node]
		D3[Hash4]
        D4[Sum: 3]
	end
	subgraph EL[Sum Leaf Node]
		E1[Hash1]
        E2[Sum: 8]
	end
	Root-->|0| BE
	Root-->|1| BI
	BI-->|0| CL
	BI-->|1| CI
	CI-->|0| DL1
	CI-->|1| DL2
	BE-->EL
```

#### Sum Hex Digests

The following diagram shows the structure of the digests of the nodes within the tree in a simplified manner, again only the `sumLeafNode` objects have a `sum` field but for visualisation purposes the sum is included in all nodes.

```mermaid
graph TB
	subgraph RI[Inner Node]
		RIA["Root Hash: Hash(Digest6+Digest7+Hex(15))+Hex(15)"]
        RIB[Sum: 15]
	end
	subgraph I1[Inner Node]
		I1A["Digest7: Hash(Digest1+Digest5+Hex(8))+Hex(8)"]
        I1B[Sum: 8]
	end
	subgraph I2[Inner Node]
		I2A["Digest6: Hash(Digest3+Digest4+Hex(7))+Hex(7)"]
        I2B[Sum: 7]
	end
	subgraph L1[Sum Leaf Node]
		L1A[Path: 0b0010000]
		L1B[Value: 0x01]
        L1C[Sum: 6]
        L1D["Hash1: Hash(Path+Value+Hex(6))"]
        L1E["Digest1: Hash1+Hex(6)"]
	end
	subgraph L3[Sum Leaf Node]
		L3A[Path: 0b1010000]
		L3B[Value: 0x03]
        L3C[Sum: 3]
        L3D["Hash3: Hash(Path+Value+Hex(3))"]
        L3E["Digest3: Hash3+Hex(3)"]
	end
	subgraph L4[Sum Leaf Node]
		L4A[Path: 0b1100000]
		L4B[Value: 0x04]
        L4C[Sum: 4]
        L4D["Hash4: Hash(Path+Value+Hex(4))"]
        L4E["Digest4: Hash4+Hex(4)"]
	end
	subgraph E1[Extension Node]
		E1A[Path: 0b01100101]
		E1B["Path Bounds: [2, 6)"]
        E1C[Sum: 5]
        E1D["Hash5: Hash(Path+PathBounds+Digest2+Hex(5))"]
        E1E["Digest5: Hash5+Hex(5)"]
	end
	subgraph L2[Sum Leaf Node]
		L2A[Path: 0b01100101]
		L2B[Value: 0x02]
        L2C[Sum: 5]
        L2D["Hash2: Hash(Path+Value+Hex(5))+Hex(5)"]
        L2E["Digest2: Hash2+Hex(5)"]
	end
	RI -->|0| I1
	RI -->|1| I2
	I1 -->|0| L1
	I1 -->|1| E1
	E1 --> L2
	I2 -->|0| L3
	I2 -->|1| L4
```

## Sum

The `Sum()` function adds the functionality to easily return the total sum of the entire tree, returned in its `uint64` format.

## Example

```go
package main

import (
	"crypto/sha256"
	"fmt"

	"github.com/pokt-network/smt"
)

func main() {
	// Initialise a new key-value store to store the nodes of the tree
	// (Note: the tree only stores hashed values, not raw value data)
	nodeStore := smt.NewSimpleMap()

	// Initialise the tree
	tree := smt.NewSparseMerkleSumTree(nodeStore, sha256.New())

	// Update tree with keys, values and their sums
	_ = tree.Update([]byte("foo"), []byte("oof"), 10)
	_ = tree.Update([]byte("baz"), []byte("zab"), 7)
	_ = tree.Update([]byte("bin"), []byte("nib"), 3)

	sum, _ := tree.Sum()
	fmt.Println(sum) // 20

	// Generate a Merkle proof for "foo"
	proof, _ := tree.Prove([]byte("foo"))
	root := tree.Root() // We also need the current tree root for the proof

	// Verify the Merkle proof for "foo"="oof" where "foo" has a sum of 10
	if valid, _ := smt.VerifySumProof(proof, root, []byte("foo"), []byte("oof"), 10, tree.Spec()); valid {
		fmt.Println("Proof verification succeeded.")
	} else {
		fmt.Println("Proof verification failed.")
	}
}

```

[plasma core docs]: https://plasma-core.readthedocs.io/en/latest/specs/sum-tree.html
