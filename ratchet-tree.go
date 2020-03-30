package mls

import (
	"fmt"
	"reflect"

	"github.com/bifurcation/mint/syntax"
)

type NodeType uint8

const (
	NodeTypeLeaf   NodeType = 0x00
	NodeTypeParent NodeType = 0x01
)

///
/// ParentNode
///

type ParentNode struct {
	PublicKey      HPKEPublicKey
	UnmergedLeaves []leafIndex     `tls:"head=4"`
	ParentHash     []byte          `tls:"head=1"`
	PrivateKey     *HPKEPrivateKey `tls:"omit"`
}

func (n *ParentNode) Equals(other *ParentNode) bool {
	pubKey := reflect.DeepEqual(n.PublicKey, other.PublicKey)
	unmerged := reflect.DeepEqual(n.UnmergedLeaves, other.UnmergedLeaves)
	parentHash := reflect.DeepEqual(n.ParentHash, other.ParentHash)

	return pubKey && unmerged && parentHash
}

func (n ParentNode) Clone() ParentNode {
	next := ParentNode{
		PublicKey:      n.PublicKey,
		UnmergedLeaves: make([]leafIndex, len(n.UnmergedLeaves)),
		ParentHash:     dup(n.ParentHash),
		PrivateKey:     n.PrivateKey,
	}

	for i, n := range n.UnmergedLeaves {
		next.UnmergedLeaves[i] = n
	}

	return next
}

func (n *ParentNode) AddUnmerged(l leafIndex) {
	n.UnmergedLeaves = append(n.UnmergedLeaves, l)
}

func (n *ParentNode) SetPublicKey(pub HPKEPublicKey) {
	n.PublicKey = pub
	n.UnmergedLeaves = []leafIndex{}
}

func (n *ParentNode) SetPrivateKey(priv HPKEPrivateKey) {
	n.PrivateKey = &priv
	n.SetPublicKey(priv.PublicKey)
}

func (n *ParentNode) SetSecret(suite CipherSuite, pathSecret []byte) error {
	priv, err := suite.hpke().Derive(pathSecret)
	if err != nil {
		return err
	}

	n.SetPrivateKey(priv)
	return nil
}

///
/// Node
///
type Node struct {
	Leaf   *KeyPackage
	Parent *ParentNode
	Hash   []byte
}

func (n *Node) Equals(other *Node) bool {
	if n == nil || other == nil {
		return n == other
	}

	switch n.Type() {
	case NodeTypeLeaf:
		return n.Leaf.Equals(*other.Leaf)
	case NodeTypeParent:
		return n.Parent.Equals(other.Parent)
	default:
		return false
	}
}

func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}

	next := &Node{}
	switch n.Type() {
	case NodeTypeLeaf:
		clone := n.Leaf.Clone()
		next.Leaf = &clone
	case NodeTypeParent:
		clone := n.Parent.Clone()
		next.Parent = &clone
	default:
		panic("Malformed node")
	}

	return next
}

func (n Node) Type() NodeType {
	switch {
	case n.Leaf != nil:
		return NodeTypeLeaf
	case n.Parent != nil:
		return NodeTypeParent
	default:
		panic("Malformed node")
	}
}

func (n Node) PublicKey() HPKEPublicKey {
	switch n.Type() {
	case NodeTypeLeaf:
		return n.Leaf.InitKey
	case NodeTypeParent:
		return n.Parent.PublicKey
	default:
		panic("Malformed node")
	}
}

func (n *Node) SetPrivateKey(priv HPKEPrivateKey) {
	switch n.Type() {
	case NodeTypeLeaf:
		n.Leaf.SetPrivateKey(priv)
	case NodeTypeParent:
		n.Parent.SetPrivateKey(priv)
	default:
		panic("Malformed node")
	}
}

func (n Node) PrivateKey() (HPKEPrivateKey, bool) {
	var maybePriv *HPKEPrivateKey
	switch n.Type() {
	case NodeTypeLeaf:
		maybePriv = n.Leaf.privateKey
	case NodeTypeParent:
		maybePriv = n.Parent.PrivateKey
	default:
		panic("Malformed node")
	}

	if maybePriv == nil {
		return HPKEPrivateKey{}, false
	}
	return *maybePriv, true
}

func (n Node) MarshalTLS() ([]byte, error) {
	s := NewWriteStream()
	nodeType := n.Type()
	err := s.Write(nodeType)
	if err != nil {
		return nil, err
	}

	switch nodeType {
	case NodeTypeLeaf:
		err = s.Write(n.Leaf)
	case NodeTypeParent:
		err = s.Write(n.Parent)
	default:
		err = fmt.Errorf("mls.node: Invalid node type")
	}
	if err != nil {
		return nil, err
	}

	return s.Data(), nil
}

func (n *Node) UnmarshalTLS(data []byte) (int, error) {
	s := NewReadStream(data)
	var nodeType NodeType
	_, err := s.Read(&nodeType)
	if err != nil {
		return 0, err
	}

	switch nodeType {
	case NodeTypeLeaf:
		n.Leaf = new(KeyPackage)
		_, err = s.Read(n.Leaf)
	case NodeTypeParent:
		n.Parent = new(ParentNode)
		_, err = s.Read(n.Parent)
	default:
		err = fmt.Errorf("mls.node: Invalid node type")
	}
	if err != nil {
		return 0, err
	}

	return s.Position(), nil
}

///
/// OptionalNode
///
type OptionalNode struct {
	Node *Node  `tls:"optional"`
	Hash []byte `tls:"omit"`
}

func newLeafNode(keyPkg KeyPackage) OptionalNode {
	return OptionalNode{Node: &Node{Leaf: &keyPkg}}
}

func newParentNode(suite CipherSuite, pathSecret []byte) (OptionalNode, error) {
	parent := ParentNode{
		UnmergedLeaves: []leafIndex{},
		ParentHash:     []byte{},
	}
	err := parent.SetSecret(suite, pathSecret)
	if err != nil {
		return OptionalNode{}, err
	}

	return OptionalNode{Node: &Node{Parent: &parent}}, nil
}

func (n OptionalNode) Clone() OptionalNode {
	return OptionalNode{
		Node: n.Node.Clone(),
		Hash: dup(n.Hash),
	}
}

func (n OptionalNode) Blank() bool {
	return n.Node == nil
}

func (n *OptionalNode) SetToBlank() {
	n.Node = nil
}

func (n *OptionalNode) MergePublic(pub HPKEPublicKey) {
	if n.Node != nil && n.Node.Type() != NodeTypeParent {
		panic("MergePublic on leaf node")
	}

	if n.Blank() {
		n.Node = &Node{Parent: &ParentNode{
			UnmergedLeaves: []leafIndex{},
			ParentHash:     []byte{},
		}}
	}
	n.Node.Parent.SetPublicKey(pub)
}

func (n *OptionalNode) setHash(suite CipherSuite, input interface{}) error {
	data, err := syntax.Marshal(input)
	if err != nil {
		return err
	}

	n.Hash = suite.digest(data)
	return nil
}

type LeafNodeHashInput struct {
	LeafIndex  leafIndex
	KeyPackage *KeyPackage `tls:"optional"`
}

func (n *OptionalNode) SetLeafHash(suite CipherSuite, index leafIndex) error {
	input := LeafNodeHashInput{
		LeafIndex:  index,
		KeyPackage: nil,
	}

	if !n.Blank() {
		if n.Node.Type() != NodeTypeLeaf {
			return fmt.Errorf("mls.rtn: SetLeafHash on non-leaf node")
		}

		input.KeyPackage = n.Node.Leaf
	}

	return n.setHash(suite, input)
}

type ParentNodeHashInput struct {
	NodeIndex  nodeIndex
	ParentNode *ParentNode `tls:"optional"`
	LeftHash   []byte      `tls:"head=1"`
	RightHash  []byte      `tls:"head=1"`
}

func (n *OptionalNode) SetParentHash(suite CipherSuite, index nodeIndex, left, right []byte) error {
	input := ParentNodeHashInput{
		NodeIndex:  index,
		ParentNode: nil,
		LeftHash:   left,
		RightHash:  right,
	}

	if !n.Blank() {
		if n.Node.Type() != NodeTypeParent {
			return fmt.Errorf("mls.rtn: SetParentHash on non-leaf node")
		}

		input.ParentNode = n.Node.Parent
	}

	return n.setHash(suite, input)
}

///
/// RatchetTree
///
type RatchetTree struct {
	Suite CipherSuite    `tls:"omit"`
	Nodes []OptionalNode `tls:"head=4"`
}

func (t *RatchetTree) dump(label string) {
	fmt.Printf("===== %s =====\n", label)
	for i := range t.Nodes {
		hash := t.Nodes[i].Hash
		if len(hash) > 4 {
			hash = hash[:4]
		}

		fmt.Printf("%4d ", i)
		if t.Nodes[i].Blank() {
			fmt.Printf("- {%x} [%x]\n", []byte{}, hash)
			continue
		}

		_, hasPriv := t.Nodes[i].Node.PrivateKey()
		pub := t.Nodes[i].Node.PublicKey().Data[:4]
		fmt.Printf("%v {%x} [%x]\n", hasPriv, pub, hash)

	}
}

func NewRatchetTree(suite CipherSuite) *RatchetTree {
	return &RatchetTree{Suite: suite}
}

func (t *RatchetTree) SetLeaf(index leafIndex, keyPkg KeyPackage) {
	n := toNodeIndex(index)
	t.Nodes[n].Node.Leaf = &keyPkg
}

func (t *RatchetTree) AddLeaf(index leafIndex, keyPkg KeyPackage) error {
	n := toNodeIndex(index)

	for len(t.Nodes) < int(n)+1 {
		t.Nodes = append(t.Nodes, OptionalNode{})
	}

	t.Nodes[n] = newLeafNode(keyPkg)

	// update unmerged list
	dp := dirpath(n, t.size())
	for _, v := range dp {
		if v == toNodeIndex(index) || t.Nodes[v].Node == nil {
			continue
		}
		t.Nodes[v].Node.Parent.AddUnmerged(index)
	}

	return t.setHashPath(index)
}

func (t *RatchetTree) UpdateLeaf(index leafIndex, keyPkg KeyPackage) error {
	n := toNodeIndex(index)
	if t.Nodes[n].Blank() {
		return fmt.Errorf("Update to unoccupied node")
	}

	t.BlankPath(index)
	t.Nodes[n] = newLeafNode(keyPkg)

	return t.setHashPath(index)
}

func (t *RatchetTree) SetLeafPrivateKey(index leafIndex, priv HPKEPrivateKey) error {
	ni := toNodeIndex(index)
	if t.Nodes[ni].Blank() {
		return fmt.Errorf("Attempt to set private key for a blank node")
	}

	t.Nodes[ni].Node.Leaf.SetPrivateKey(priv)
	return nil
}

func (t RatchetTree) PathSecrets(start nodeIndex, pathSecret []byte) map[nodeIndex][]byte {
	secrets := map[nodeIndex][]byte{}

	curr := start
	next := parent(curr, t.size())
	secrets[curr] = dup(pathSecret)

	for curr != t.rootIndex() {
		secrets[next] = t.pathStep(secrets[curr])
		curr = next
		next = parent(curr, t.size())
	}

	return secrets
}

func (t *RatchetTree) Encap(from leafIndex, context, leafSecret []byte) (DirectPath, []byte, error) {
	// list of updated nodes - output
	leafNode := toNodeIndex(from)
	dp := DirectPath{}

	// generate the necessary path secrets
	secrets := t.PathSecrets(leafNode, leafSecret)

	cp := copath(leafNode, t.size())
	for _, v := range cp {
		parent := parent(v, t.size())
		if parent == leafNode {
			continue
		}

		// update the non-updated child's parent with the newly
		// computed path-secret
		pathSecret := secrets[parent]
		n, err := newParentNode(t.Suite, pathSecret)
		if err != nil {
			return DirectPath{}, nil, err
		}
		t.Nodes[parent] = n

		//update nodes on the direct path to share it with others
		pathNode := DirectPathNode{PublicKey: n.Node.Parent.PublicKey}

		// encrypt the secret to resolution maintained
		res := t.resolve(v)
		for _, rnode := range res {
			pk := t.Nodes[rnode].Node.PublicKey()
			ct, err := t.Suite.hpke().Encrypt(pk, context, pathSecret)
			if err != nil {
				return DirectPath{}, nil, err
			}
			pathNode.EncryptedPathSecrets = append(pathNode.EncryptedPathSecrets, ct)
		}

		dp.Nodes = append(dp.Nodes, pathNode)
	}

	err := t.setHashPath(from)
	if err != nil {
		return DirectPath{}, nil, err
	}

	return dp, secrets[t.rootIndex()], nil
}

func (t *RatchetTree) ImplantFrom(from, to leafIndex, pathSecret []byte) ([]byte, error) {
	return t.Implant(ancestor(from, to), pathSecret)
}

func (t *RatchetTree) Implant(start nodeIndex, pathSecret []byte) ([]byte, error) {
	secrets := t.PathSecrets(start, pathSecret)

	for curr, secret := range secrets {
		node, err := newParentNode(t.Suite, secret)
		if err != nil {
			return nil, err
		}

		if t.Nodes[curr].Blank() || t.Nodes[curr].Node.Type() != NodeTypeParent {
			return nil, fmt.Errorf("Attempt to implant invalid node %v", curr)
		}

		if !t.Nodes[curr].Node.Parent.PublicKey.Equals(node.Node.Parent.PublicKey) {
			return nil, fmt.Errorf("Incorrect secret for existing public key")
		}

		t.Nodes[curr].Node.Parent.SetPrivateKey(*node.Node.Parent.PrivateKey)
	}

	// XXX(rlb): Set root secret?
	return secrets[t.rootIndex()], nil
}

func (t *RatchetTree) decryptPathSecret(from leafIndex, context []byte, path DirectPath) (nodeIndex, []byte, error) {
	cp := copath(toNodeIndex(from), t.size())
	if len(path.Nodes) != len(cp) {
		return 0, nil, fmt.Errorf("mls.rtn: Malformed (cp) DirectPath %d %d %v", len(path.Nodes), len(cp)+1, cp)
	}

	for i, curr := range cp {
		res := t.resolve(curr)
		pathNode := path.Nodes[i]

		if len(pathNode.EncryptedPathSecrets) != len(res) {
			return 0, nil, fmt.Errorf("mls.rtn: Malformed Ratchet Node")
		}

		for idx, v := range res {
			if t.Nodes[v].Blank() {
				continue
			}

			priv, ok := t.Nodes[v].Node.PrivateKey()
			if !ok {
				continue
			}

			encryptedSecret := pathNode.EncryptedPathSecrets[idx]
			pathSecret, err := t.Suite.hpke().Decrypt(priv, context, encryptedSecret)
			if err != nil {
				return 0, nil, fmt.Errorf("mls:rtn: Ratchet node %v Decryption failure %v", v, err)
			}

			parentNode := parent(curr, t.size())
			return parentNode, pathSecret, nil
		}
	}

	return 0, nil, fmt.Errorf("mls:rtn: No private key available for decrypt")
}

func (t *RatchetTree) Decap(from leafIndex, context []byte, path DirectPath) ([]byte, error) {
	// Set public keys
	dp := dirpath(toNodeIndex(from), t.size())
	if len(path.Nodes) != len(dp) {
		return nil, fmt.Errorf("mls.rtn: Malformed (dp) DirectPath %d %d", len(path.Nodes), len(dp))
	}

	for i, node := range dp {
		t.Nodes[node].MergePublic(path.Nodes[i].PublicKey)
	}

	// Decrypt and implant path secret
	overlap, pathSecret, err := t.decryptPathSecret(from, context, path)
	if err != nil {
		return nil, err
	}

	rootSecret, err := t.Implant(overlap, pathSecret)
	if err != nil {
		return nil, err
	}

	err = t.setHashPath(from)
	if err != nil {
		return nil, err
	}

	return rootSecret, nil
}

func (t *RatchetTree) BlankPath(index leafIndex) error {
	if len(t.Nodes) == 0 {
		return nil
	}

	lc := t.size()
	r := t.rootIndex()
	leaf := toNodeIndex(index)
	for curr := leaf; curr != r; curr = parent(curr, lc) {
		t.Nodes[curr].SetToBlank()
	}

	t.Nodes[r].SetToBlank()

	return t.setHashPath(index)
}

func (t RatchetTree) KeyPackage(index leafIndex) (KeyPackage, bool) {
	ni := toNodeIndex(index)
	if t.Nodes[ni].Blank() {
		return KeyPackage{}, false
	}

	return *t.Nodes[ni].Node.Leaf, true
}

func (t RatchetTree) RootHash() []byte {
	r := root(t.size())
	return t.Nodes[r].Hash
}

func (t RatchetTree) Clone() RatchetTree {
	next := RatchetTree{
		Suite: t.Suite,
		Nodes: make([]OptionalNode, len(t.Nodes)),
	}

	for i, n := range t.Nodes {
		next.Nodes[i] = n.Clone()
	}

	return next
}

func (t RatchetTree) Equals(o RatchetTree) bool {
	if len(t.Nodes) != len(o.Nodes) {
		return false
	}

	for i := 0; i < len(t.Nodes); i++ {
		if !t.Nodes[i].Node.Equals(o.Nodes[i].Node) {
			return false
		}
	}
	return true
}

func (t RatchetTree) LeftmostFree() leafIndex {
	curr := leafIndex(0)
	size := leafIndex(t.size())
	for {
		if curr < size && !t.Nodes[toNodeIndex(curr)].Blank() {
			curr++
		} else {
			break
		}
	}
	return curr
}

func (t RatchetTree) Find(kp KeyPackage) (leafIndex, bool) {
	num := t.size()
	for i := leafIndex(0); leafCount(i) < num; i++ {
		ni := toNodeIndex(i)
		n := t.Nodes[ni]
		if n.Blank() {
			continue
		}

		if n.Node.Leaf.Equals(kp) {
			return i, true
		}
	}

	return 0, false
}

//// Ratchet Tree helpers functions

// number of leaves in the ratchet tree
func (t RatchetTree) size() leafCount {
	return leafWidth(nodeCount(len(t.Nodes)))
}

func (t RatchetTree) rootIndex() nodeIndex {
	return root(t.size())
}

func (t RatchetTree) pathStep(pathSecret []byte) []byte {
	ps := t.Suite.hkdfExpandLabel(pathSecret, "path", []byte{}, t.Suite.constants().SecretSize)
	return ps
}

func (t RatchetTree) resolve(index nodeIndex) []nodeIndex {
	// Resolution of non-blank is node + unmerged leaves
	if !t.Nodes[index].Blank() {
		res := []nodeIndex{index}
		if level(index) > 0 {
			for _, v := range t.Nodes[index].Node.Parent.UnmergedLeaves {
				res = append(res, nodeIndex(v))
			}
		}
		return res
	}

	// Resolution of blank leaf is the empty list
	if level(index) == 0 {
		return []nodeIndex{}
	}

	// Resolution of blank intermediate node is concatenation of the resolutions
	// of the children
	l := t.resolve(left(index))
	r := t.resolve(right(index, t.size()))
	l = append(l, r...)
	return l
}

func (t *RatchetTree) setHash(index nodeIndex) error {
	if level(index) == 0 {
		return t.Nodes[index].SetLeafHash(t.Suite, toLeafIndex(index))
	}

	lh := t.Nodes[left(index)].Hash
	rh := t.Nodes[right(index, t.size())].Hash
	return t.Nodes[index].SetParentHash(t.Suite, index, lh, rh)
}

func (t *RatchetTree) setHashPath(index leafIndex) error {
	curr := toNodeIndex(index)

	size := t.size()
	r := root(size)
	for {
		err := t.setHash(curr)
		if err != nil {
			return err
		}

		if curr == r {
			break
		}

		curr = parent(curr, size)
	}

	return nil
}

func (t *RatchetTree) setHashSubtree(index nodeIndex) error {
	if len(t.Nodes) == 0 {
		return nil
	}

	if level(index) == 0 {
		return t.setHash(index)
	}

	l := left(index)
	err := t.setHashSubtree(l)
	if err != nil {
		return err
	}

	r := right(index, t.size())
	err = t.setHashSubtree(r)
	if err != nil {
		return err
	}

	return t.setHash(index)
}

func (t *RatchetTree) SetHashAll() error {
	return t.setHashSubtree(root(t.size()))
}

// Isolated getters and setters for secret state
type TreeSecrets struct {
	PrivateKeys map[nodeIndex]HPKEPrivateKey `tls:"head=4"`
}

func (t *RatchetTree) SetSecrets(ts TreeSecrets) {
	for ix, priv := range ts.PrivateKeys {
		t.Nodes[ix].Node.SetPrivateKey(priv)
	}
}

func (t RatchetTree) GetSecrets() TreeSecrets {
	ts := TreeSecrets{
		PrivateKeys: map[nodeIndex]HPKEPrivateKey{},
	}

	for i, n := range t.Nodes {
		if n.Blank() {
			continue
		}

		priv, ok := n.Node.PrivateKey()
		if !ok {
			continue
		}

		ts.PrivateKeys[nodeIndex(i)] = priv
	}

	return ts
}
