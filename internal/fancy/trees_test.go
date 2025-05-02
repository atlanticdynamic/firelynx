package fancy_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// TreesTestSuite is a test suite for testing tree-related functionality
type TreesTestSuite struct {
	suite.Suite
}

// TestNewComponentTree tests the creation of a new component tree
func (s *TreesTestSuite) TestNewComponentTree() {
	title := "Test Component"
	
	// Create a new component tree
	compTree := fancy.NewComponentTree(title)
	
	// Assert the tree was created
	assert.NotNil(s.T(), compTree)
	
	// Check that we can get the underlying tree
	t := compTree.Tree()
	assert.NotNil(s.T(), t)
	
	// Verify the tree has the title as root
	assert.Contains(s.T(), t.String(), title)
}

// TestAddBranch tests adding a branch to a component tree
func (s *TreesTestSuite) TestAddBranch() {
	compTree := fancy.NewComponentTree("Root")
	branchText := "Branch 1"
	
	// Add a branch
	branch := compTree.AddBranch(branchText)
	
	// Assert the branch was created
	assert.NotNil(s.T(), branch)
	assert.IsType(s.T(), &tree.Tree{}, branch)
	
	// Verify the branch text appears in the rendered tree
	treeString := compTree.Tree().String()
	assert.Contains(s.T(), treeString, branchText)
}

// TestAddChild tests adding a child node to the component tree
func (s *TreesTestSuite) TestAddChild() {
	compTree := fancy.NewComponentTree("Root")
	childText := "Child Node"
	
	// Add a child
	child := compTree.AddChild(childText)
	
	// Assert the child was created
	assert.NotNil(s.T(), child)
	assert.IsType(s.T(), &tree.Tree{}, child)
	
	// Verify the child text appears in the rendered tree
	treeString := compTree.Tree().String()
	assert.Contains(s.T(), treeString, childText)
}

// TestEndpointTree tests creating a tree for endpoint visualization
func (s *TreesTestSuite) TestEndpointTree() {
	endpointID := "test-endpoint"
	
	// Create an endpoint tree
	endpointTree := fancy.EndpointTree(endpointID)
	
	// Assert the tree was created
	assert.NotNil(s.T(), endpointTree)
	
	// Verify the tree has the styled endpoint ID
	// The exact string comparison is tricky because of ANSI styling,
	// but we can check if the ID is part of the rendered string
	treeString := endpointTree.Tree().String()
	assert.Contains(s.T(), treeString, endpointID)
}

// TestRouteTree tests creating a tree for route visualization
func (s *TreesTestSuite) TestRouteTree() {
	routeInfo := "/api/v1/test"
	
	// Create a route tree
	routeTree := fancy.RouteTree(routeInfo)
	
	// Assert the tree was created
	assert.NotNil(s.T(), routeTree)
	
	// Verify the tree has the styled route info
	treeString := routeTree.Tree().String()
	assert.Contains(s.T(), treeString, routeInfo)
}

// TestTreeChaining tests the ability to chain tree operations
func (s *TreesTestSuite) TestTreeChaining() {
	compTree := fancy.NewComponentTree("Root")
	
	// Create a more complex tree with chained operations
	branch1 := compTree.AddBranch("Branch 1")
	branch1.Child("Child 1.1")
	branch1.Child("Child 1.2")
	
	branch2 := compTree.AddBranch("Branch 2")
	branch2.Child("Child 2.1")
	
	// Verify the final tree structure contains all the elements
	treeString := compTree.Tree().String()
	assert.Contains(s.T(), treeString, "Root")
	assert.Contains(s.T(), treeString, "Branch 1")
	assert.Contains(s.T(), treeString, "Child 1.1")
	assert.Contains(s.T(), treeString, "Child 1.2")
	assert.Contains(s.T(), treeString, "Branch 2")
	assert.Contains(s.T(), treeString, "Child 2.1")
}

// TestStyleConsistency tests that the styles are consistently applied
func (s *TreesTestSuite) TestStyleConsistency() {
	// Test endpoint and route trees use the correct styles
	endpointTree := fancy.EndpointTree("endpoint")
	routeTree := fancy.RouteTree("/path")
	
	// Each tree should have a different rendered string due to different styles
	assert.NotEqual(s.T(), endpointTree.Tree().String(), routeTree.Tree().String())
}

// Run the test suite
func TestTreesSuite(t *testing.T) {
	suite.Run(t, new(TreesTestSuite))
}
