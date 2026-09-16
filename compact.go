package smt

// Compaction of persisted leaves.
//
// With a nil value hasher the leaf holds the RAW value, so a resident trie
// costs roughly len(value) per leaf even though Commit already wrote those
// exact bytes to the node store, keyed by the leaf's own digest. Compaction
// drops that redundant in-memory copy and resolves it back from the store on
// the few paths that actually need a value.
//
// The invariant a compacted leaf asserts is:
//
//	compacted => persisted && digest != nil && valueHash == nil
//
// which reads as "the bytes are in the node store under digest". That is a
// claim about the store, not the absence of a field, which is why it needs an
// explicit flag rather than a nil check: a plain SMT with a nil value hasher
// can legitimately hold an empty valueHash.

// CompactPersistedLeaves drops the in-memory value of every leaf that has
// already been written to the node store, and returns the number of leaves
// compacted.
//
// It only walks nodes that are resident in memory: a lazyNode is by definition
// not resident, so there is nothing to free below it and resolving it would
// cost the very memory this is meant to reclaim.
//
// Leaves that are not yet persisted are left untouched. Compacting one would
// leave a leaf whose value is in neither memory nor the store, and whose digest
// would then be computed over a truncated preimage.
//
// What "persisted" means, and what the caller owes. Compaction trusts each
// node's persisted flag, and commit sets that flag once the store's Set has
// returned without error. That records that the store accepted the node, not
// that it holds it: a store that accepts Set into a buffer and writes it out
// later reports success from Set even when that later write fails, and nothing
// in the trie can observe that failure. Leaves affected that way must not be
// compacted until they have been written again, for example by updating the
// same key and committing successfully; compacting them first drops the only
// remaining copy of their value. Only the caller can see the store fail, so
// only the caller can hold compaction back. Run this after a Commit that
// succeeded and whose writes the store has confirmed, never from inside Commit.
//
// Cost. It is meant to be called after every Commit. A pass that walked the
// whole trie every time would be O(N) per call and O(N^2) across N updates:
// measured at 20k leaves that is already 3.5 s, and it grows with the square.
// So a node whose whole subtree is compacted is marked, and later passes stop
// there, which makes a pass cost the depth of the path that changed rather
// than the size of the trie. The marks ride on setDirty, which every mutation
// already calls on the way back up.
//
// Concurrency. Compaction writes to the trie, and so does every read that has
// to bring a value back: Get, Prove and ProveClosest restore a compacted leaf's
// value from the store and record that they did, and resolving a lazy node is
// recorded too. None of these is safe to call concurrently with any other
// method without external locking. Get was already unsafe that way on a trie
// holding lazy nodes, because it replaces a resolved node in place; compaction
// extends the same caveat to Prove and ProveClosest, and to fully resident
// tries.
//
// The error is in the signature so that a future pass can report a failure
// without an API change. This implementation never returns a non-nil error.
func (smt *SMT) CompactPersistedLeaves() (int, error) {
	// Anything pulled in from the store arrived outside the setDirty path, so
	// the marks cannot be trusted and this pass walks everything.
	force := smt.resolvedSinceCompaction
	compacted, _ := smt.compactNode(smt.root, force)
	smt.resolvedSinceCompaction = false
	return compacted, nil
}

// compactNode recursively compacts the resident subtree rooted at node. It
// returns how many leaves it compacted, and whether every resident leaf below
// node now holds no value.
func (smt *SMT) compactNode(node trieNode, force bool) (compacted int, allCompacted bool) {
	switch n := node.(type) {
	case *leafNode:
		if n.compacted {
			return 0, true
		}
		if !n.persisted || n.digest == nil {
			// The value is in neither the store nor anywhere else. Dropping it
			// would leave a digest computed over a truncated preimage, which
			// commit would then persist.
			return 0, false
		}
		// Both path and digest may be subslices of a larger buffer: a leaf
		// resolved from the store keeps the whole encoded node alive through
		// path (see parseLeafNode), and its digest points into the parent's
		// encoded bytes. Copying them is what actually frees the value.
		n.path = cloneBytes(n.path)
		n.digest = cloneBytes(n.digest)
		n.valueHash = nil
		n.compacted = true
		return 1, true

	case *innerNode:
		if n.compactedSubtree && !force {
			return 0, true
		}
		left, leftAll := smt.compactNode(n.leftChild, force)
		right, rightAll := smt.compactNode(n.rightChild, force)
		n.compactedSubtree = leftAll && rightAll
		return left + right, n.compactedSubtree

	case *extensionNode:
		if n.compactedSubtree && !force {
			return 0, true
		}
		count, all := smt.compactNode(n.child, force)
		n.compactedSubtree = all
		return count, all

	default:
		// nil and lazyNode: nothing resident to compact, and nothing below
		// them holds a value in memory either.
		return 0, true
	}
}

// resolveLeafValue restores the value of a compacted leaf from the node store.
// It is a no-op for any leaf that still holds its value.
//
// The leaf stays hydrated afterwards, so repeated reads of the same leaf cost
// one store read, not one per read. A later CompactPersistedLeaves drops it
// again.
func (smt *SMT) resolveLeafValue(leaf *leafNode) error {
	if !leaf.compacted {
		return nil
	}
	data, err := smt.nodes.Get(leaf.digest)
	if err != nil {
		return err
	}
	// parseLeafNode returns subslices of data; copy so the leaf does not retain
	// the whole encoded node on top of the value.
	_, valueHash := smt.parseLeafNode(data)
	leaf.valueHash = cloneBytes(valueHash)
	leaf.compacted = false
	// This leaf now holds a value again, but its ancestors still carry a
	// compactedSubtree mark that would make the next pass skip it.
	smt.resolvedSinceCompaction = true
	return nil
}

// resolveCompactedLeaf restores the value of node when it is a compacted leaf,
// so that the node can be encoded. Any other node type is left alone.
func (smt *SMT) resolveCompactedLeaf(node trieNode) error {
	leaf, ok := node.(*leafNode)
	if !ok {
		return nil
	}
	return smt.resolveLeafValue(leaf)
}

// assertNotCompacted panics if a compacted leaf is about to be encoded.
//
// Encoding a compacted leaf would silently produce a preimage of prefix+path
// with no value, whose digest would then be computed over the wrong bytes and,
// through commit, written to the store. Failing loudly here is what keeps that
// from becoming a corrupt root nobody notices.
func assertNotCompacted(leaf *leafNode) {
	if leaf.compacted {
		panic("smt: encoding a compacted leaf; its value must be resolved first")
	}
}

// cloneBytes returns a copy of b that shares no backing array with it.
func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
