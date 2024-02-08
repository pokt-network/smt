# Sparse Merkle Sum Trie (smst)

<!-- toc -->

- [Sparse Merkle Sum Trie (smst)](#sparse-merkle-sum-trie-smst)
  - [Overview](#overview)
  - [Implementation](#implementation)
    - [Sum Encoding](#sum-encoding)
    - [Digests](#digests)
    - [Visualisations](#visualisations)
      - [General Trie Structure](#general-trie-structure)
      - [Binary Sum Digests](#binary-sum-digests)
  - [Sum](#sum)
  - [Roots](#roots)
  - [Nil Values](#nil-values)

<!-- tocstop -->

## Overview

Merkle Sum tries function very similarly to regular Merkle tries, with the
primary difference being that each leaf node in a Merkle sum trie includes a
`sum` in addition to its value. This allows for the entire trie's total sum to
be calculated easily, as the sum of any branch is the sum of its children. Thus
the sum of the root node is the sum of the entire trie. Like a normal Merkle
trie, the Merkle sum trie allows for the efficient verification of its members,
proving inclusion/exclusion of certain elements and generally functions the same.

Merkle sum tries can be very useful for blockchain applications in that they can
easily track accounts balances and, thus, the total balance of all accounts.
They can be very useful in proof of reserve systems whereby one needs to prove
the inclusion of an element that is a component of the total sum, along with a
verifiable total sum of all elements.

## Implementation

The implementation of the Sparse Merkle Sum Trie (SMST) follows, in principle,
the same implementation as the [Plasma Core Merkle Sum Tree][plasma core docs].
The main differences with the current SMT implementation are outlined below.
The primary difference lies in the encoding of node data within the trie to
accommodate for the sum.

_NOTE: The Plasma Core Merkle Sum trie uses a 16 byte hex string to encode the
sum whereas this SMST implementation uses an 8 byte binary representation of the
`uint64` sum._

In practice the SMST is a wrapper around the SMT with a new field added to the
`TrieSpec`: `sumTrie bool` this determines whether the SMT should follow its
regular encoding of that of the sum trie.

See: the [SMT documentation](./smt.md) for the details on how the SMT works.

The majority of the code relating to the SMST can be found in:

- [smst.go](../smst.go) - main SMT wrapper functionality
- [hasher.go](../hasher.go) - SMST encoding functions
- [types.go](../types.go) - SMST interfaces and node serialisation/hashing
  functions

### Sum Encoding

The sum for any node is encoded in a byte array with a fixed size (`[8]byte`)
this allows for the sum to fully represent a `uint64` value in binary form.
The golang `encoding/binary` package is used to encode the sum with
`binary.BigEndian.PutUint64(sumBz[:], sum)` into a byte array `sumBz`.

In order for the SMST to include the sum into a leaf node the SMT the SMST
initialises the SMT with the `WithValueHasher(nil)` option so that the SMT does
**not** hash any values. The SMST will then hash the value and append the sum
bytes to the end of the hashed value, using whatever `ValueHasher` was given to
the SMST on initialisation.

```mermaid
graph TD
	subgraph KVS[Key-Value-Sum]
		K1["Key: foo"]
		K2["Value: bar"]
		K3["Sum: 10"]
	end
	subgraph SMST[SMST]
		SS1[ValueHasher: SHA256]
		subgraph SUM["SMST.Update()"]
			SU1["valueHash = ValueHasher(Value)"]
			SU2["sumBytes = binary(Sum)"]
			SU3["valueHash = append(valueHash, sumBytes...)"]
		end
	end
	subgraph SMT[SMT]
		SM1[ValueHasher: nil]
		subgraph UPD["SMT.Update()"]
			U2["SMT.nodeStore.Set(Key, valueHash)"]
		end
	end
	KVS --"Key + Value + Sum"--> SMST
	SMST --"Key + valueHash"--> SMT
```

### Digests

The digest for any node in the SMST is calculated in partially the same manner
as the regular SMT. The main differences are that the sum is included in the
digest `preimage` - meaning the hash of any node's data includes **BOTH** its
data _and_ sum. In addition to this the sum is appended to the hash producing
digests like so:

`digest = [node hash]+[8 byte sum]`

Therefore for the following node types, the digests are computed as follows:

- **Inner Nodes**
  - Prefix: `[]byte{1}`
  - `sumBytes = binary(leftChild.sum+rightChild.sum)`
  - `digest = hash([]byte{1} + leftChild.digest + rightChild.digest + sumBytes) + sumBytes`
- **Extension Nodes**
  - Prefix: `[]byte{2}`
  - `sumBytes = binary(child.sum)`
  - `digest = hash([]byte{2} + pathBounds + path + child.digest + sumBytes) + sumBytes`
- **Leaf Nodes**
  - Prefix: `[]byte{0}`
  - `sumBytes = binary(sum)`
  - `digest = hash([]byte{0} + path + valueHash) + sumBytes`
    - **Note**: as mentioned above the `valueHash` is already appended with the
      `sumBytes` prior to insertion in the underlying SMT
- **Lazy Nodes**
  - Prefix of the actual node type is stored in the persisted digest as
    determined above
  - `digest = persistedDigest`

This means that with a hasher such as `sha256.New()` whose hash size is
`32 bytes`, the digest of any node will be `40 bytes` in length.

### Visualisations

The following diagrams are representations of how the trie and its components
can be visualised.

#### General Trie Structure

None of the nodes have a different structure to the regular SMT, but the digests
of nodes now include their sum as described above and the sum is included in the
leaf node's value. For the purposes of visualization, the sum is included in all
nodes as an extra field.

```mermaid
graph TB
	subgraph Root
		A1["Digest: Hash(Hash(Path+H1)+Hash(H2+(Hash(H3+H4)))+Binary(20))+Binary(20)"]
        A2[Sum: 20]
	end
	subgraph BI[Inner Node]
		B1["Digest: Hash(H2+(Hash(H3+H4))+Binary(12))+Binary(12)"]
        B2[Sum: 12]
	end
	subgraph BE[Extension Node]
		B3["Digest: Hash(Path+H1+Binary(8))+Binary(8)"]
        B4[Sum: 8]
	end
	subgraph CI[Inner Node]
		C1["Digest: Hash(H3+H4+Binary(7))+Binary(7)"]
        C2[Sum: 7]
	end
	subgraph CL[Leaf Node]
		C3[Digest: H2]
        C4[Sum: 5]
	end
	subgraph DL1[Leaf Node]
		D1[Digest: H3]
        D2[Sum: 4]
	end
	subgraph DL2[Leaf Node]
		D3[Digest: H4]
        D4[Sum: 3]
	end
	subgraph EL[Leaf Node]
		E1[Digest:  H1]
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

#### Binary Sum Digests

The following diagram shows the structure of the digests of the nodes within
the trie in a simplified manner, again none of the nodes have a `sum` field,
but for visualisation purposes the sum is included in all nodes with the
exception of the leaf nodes where the sum is shown as part of its value.

```mermaid
graph TB
	subgraph RI[Inner Node]
		RIA["Root Hash: Hash(D6+D7+Binary(18))+Binary(18)"]
        RIB[Sum: 15]
	end
	subgraph I1[Inner Node]
		I1A["D7: Hash(D1+D5+Binary(11))+Binary(11)"]
        I1B[Sum: 11]
	end
	subgraph I2[Inner Node]
		I2A["D6: Hash(D3+D4+Binary(7))+Binary(7)"]
        I2B[Sum: 7]
	end
	subgraph L1[Leaf Node]
		L1A[Path: 0b0010000]
		L1B["Value: 0x01+Binary(6)"]
        L1C["H1: Hash(Path+Value+Binary(6))"]
        L1D["D1: H1+Binary(6)"]
	end
	subgraph L3[Leaf Node]
		L3A[Path: 0b1010000]
		L3B["Value: 0x03+Binary(3)"]
        L3C["H3: Hash(Path+Value+Binary(3))"]
        L3D["D3: H3+Binary(3)"]
	end
	subgraph L4[Leaf Node]
		L4A[Path: 0b1100000]
		L4B["Value: 0x04+Binary(4)"]
        L4C["H4: Hash(Path+Value+Binary(4))"]
        L4D["D4: H4+Binary(4)"]
	end
	subgraph E1[Extension Node]
		E1A[Path: 0b01100101]
		E1B["Path Bounds: [2, 6)"]
        E1C[Sum: 5]
        E1D["H5: Hash(Path+PathBounds+D2+Binary(5))"]
        E1E["D5: H5+Binary(5)"]
	end
	subgraph L2[Leaf Node]
		L2A[Path: 0b01100101]
		L2B["Value: 0x02+Binary(5)"]
        L2C["H2: Hash(Path+Value+Hex(5))+Binary(5)"]
        L2D["D2: H2+Binary(5)"]
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

The `Sum()` function adds functionality to easily retrieve the trie's current
sum as a `uint64`.

## Roots

The root of the tree is a slice of bytes. `MerkleRoot` is an alias for `[]byte`.
This design enables easily passing around the data (e.g. on-chain)
while maintaining primitive usage in different use cases (e.g. proofs).

`MerkleRoot` provides helpers, such as retrieving the `Sum() uint64` to
interface with data it captures.

## Nil Values

A `nil` value and `0` weight is the same as the placeholder value and default
sum in the SMST and as such inserting a key with a `nil` value has specific
behaviours. Although the insertion of a key-value-weight grouping with a `nil`
value and `0` weight will alter the root hash, a proof will not recognise the
key as being in the trie.

Assume `(key, value, weight)` groupings as follows:

- `(key, nil, 0)` -> DOES modify the `root` hash
  - Proving this `key` is in the trie will fail
- `(key, nil, weight)` -> DOES modify the `root` hash
  - Proving this `key` is in the trie will succeed
- `(key, value, 0)` -> DOES modify the `root` hash
  - Proving this `key` is in the trie will succeed
- `(key, value, weight)` -> DOES modify the `root` hash
  - Proving this `key` is in the trie will succeed
- `(key, value, weight)` -> DOES modify the `root` hash
  - Proving this `key` is in the trie will succeed

[plasma core docs]: https://plasma-core.readthedocs.io/en/latest/specs/sum-tree.html
