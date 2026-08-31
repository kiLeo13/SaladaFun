// Package accounttree constructs safe application trees from sparse Discord
// account-parent relationships.
package accounttree

import (
	"errors"
	"fmt"

	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/domain/entity"
)

var ErrInvalidTree = errors.New("invalid Discord account tree")

// Node identifies one Discord account and its direct child accounts.
type Node struct {
	UserID   uint64
	Children []*Node
}

// Tree is a complete rooted hierarchy and includes its root in Count.
type Tree struct {
	Root  *Node
	Count int
}

// Service loads and validates Discord account hierarchies.
type Service struct {
	repository Repository
}

// NewService constructs a hierarchy service backed by the supplied repository.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// Tree returns the complete hierarchy containing userID. An unlinked user is a
// one-account tree rooted at that user.
func (s *Service) Tree(userID uint64) (*Tree, error) {
	links, err := s.repository.Tree(userID)
	if err != nil {
		return nil, fmt.Errorf("load Discord account tree: %w", err)
	}
	if len(links) == 0 {
		return &Tree{Root: &Node{UserID: userID}, Count: 1}, nil
	}
	return buildTree(userID, links)
}

// buildTree converts one recursively selected hierarchy into a linked tree.
func buildTree(requestedUserID uint64, links []entity.DiscordAccountLink) (*Tree, error) {
	nodes := make(map[uint64]*Node, len(links))
	var root *Node
	for _, link := range links {
		if link.UserID == 0 {
			return nil, ErrInvalidTree
		}
		if _, duplicate := nodes[link.UserID]; duplicate {
			return nil, ErrInvalidTree
		}
		nodes[link.UserID] = &Node{UserID: link.UserID}
	}
	if _, found := nodes[requestedUserID]; !found {
		return nil, ErrInvalidTree
	}
	for _, link := range links {
		node := nodes[link.UserID]
		if link.ParentID == nil {
			if root != nil {
				return nil, ErrInvalidTree
			}
			root = node
			continue
		}
		parent := nodes[*link.ParentID]
		if parent == nil || parent == node {
			return nil, ErrInvalidTree
		}
		parent.Children = append(parent.Children, node)
	}
	if root == nil {
		return nil, ErrInvalidTree
	}
	if count, valid := countNodes(root, make(map[uint64]struct{}, len(nodes))); !valid || count != len(nodes) {
		return nil, ErrInvalidTree
	} else {
		return &Tree{Root: root, Count: count}, nil
	}
}

// countNodes verifies that every selected node is reachable exactly once.
func countNodes(node *Node, seen map[uint64]struct{}) (int, bool) {
	if _, duplicate := seen[node.UserID]; duplicate {
		return 0, false
	}
	seen[node.UserID] = struct{}{}
	count := 1
	for _, child := range node.Children {
		childCount, valid := countNodes(child, seen)
		if !valid {
			return 0, false
		}
		count += childCount
	}
	return count, true
}
