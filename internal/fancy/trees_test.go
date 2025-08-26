package fancy_test

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
)

// TestTree tests the creation of a basic tree with common styling
func TestTree(t *testing.T) {
	// Create a new tree
	tree := fancy.Tree()

	// Assert the tree was created
	assert.NotNil(t, tree)
	// The tree is of type *lipgloss.tree.Tree

	// Add some content and verify it renders
	tree.Root("Root Node")
	child := tree.Child("Child Node")
	child.Child("Grandchild")

	// Verify the tree string contains the added nodes
	treeString := tree.String()
	assert.Contains(t, treeString, "Root Node")
	assert.Contains(t, treeString, "Child Node")
	assert.Contains(t, treeString, "Grandchild")
}

// TestBranchNode tests creating a styled section header node
func TestBranchNode(t *testing.T) {
	// Create a branch node with title and count
	title := "Test Title"
	count := "(5)"
	branchNode := fancy.BranchNode(title, count)

	// Assert the branch node was created
	assert.NotNil(t, branchNode)
	// Verify it's the right type without explicit type assertion

	// Verify the tree string contains both the title and count
	treeString := branchNode.String()
	assert.Contains(t, treeString, title)
	assert.Contains(t, treeString, count)
}

// TestTruncateString tests string truncation for various cases
func TestTruncateString(t *testing.T) {
	t.Run("String shorter than maxLength", func(t *testing.T) {
		shortString := "Short string"
		maxLength := 20
		result := fancy.TruncateString(shortString, maxLength)
		assert.Equal(t, shortString, result, "Short strings should not be truncated")
	})

	t.Run("String exactly at maxLength", func(t *testing.T) {
		exactString := "Exactly twenty chars!"
		maxLength := 20
		result := fancy.TruncateString(exactString, maxLength)
		// Note: The current implementation truncates when length equals maxLength
		expected := "Exactly twenty ch..."
		assert.Equal(
			t,
			expected,
			result,
			"Strings exactly at maxLength are truncated to maxLength-3 + '...'",
		)
	})

	t.Run("String one character shorter than maxLength", func(t *testing.T) {
		almostExactString := "19 character string"
		maxLength := 20
		result := fancy.TruncateString(almostExactString, maxLength)
		assert.Equal(
			t,
			almostExactString,
			result,
			"Strings less than maxLength should not be truncated",
		)
	})

	t.Run("String longer than maxLength", func(t *testing.T) {
		longString := "This is a very long string that should be truncated"
		maxLength := 15
		result := fancy.TruncateString(longString, maxLength)
		assert.Equal(t, "This is a ve...", result, "Long strings should be truncated with ellipsis")
		assert.Len(t, result, maxLength, "Truncated string length should match maxLength")
	})

	t.Run("Empty string", func(t *testing.T) {
		emptyString := ""
		maxLength := 10
		result := fancy.TruncateString(emptyString, maxLength)
		assert.Equal(t, emptyString, result, "Empty strings should remain empty")
	})

	t.Run("MaxLength equal to ellipsis length", func(t *testing.T) {
		longString := "This is a very long string"
		maxLength := 3
		result := fancy.TruncateString(longString, maxLength)
		assert.Equal(t, "...", result, "When maxLength equals 3, should truncate to just '...'")
	})

	t.Run("MaxLength allows one character plus ellipsis", func(t *testing.T) {
		longString := "This is a very long string"
		maxLength := 4
		result := fancy.TruncateString(longString, maxLength)
		assert.Equal(t, "T...", result, "With maxLength=4, should have 1 character + ellipsis")
	})

	t.Run("Handle unsafe maxLength values", func(t *testing.T) {
		// For unsafe maxLength values, we expect a reasonable fallback behavior
		// This is testing that we don't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("The function panicked with maxLength=2: %v", r)
			}
		}()
		maxLength := 2
		_ = fancy.TruncateString("Any string", maxLength)
	})
}

// TestTreeWithStyling tests that the tree created by Tree() has the expected styling
func TestTreeWithStyling(t *testing.T) {
	// Create a new tree
	tree := fancy.Tree()

	// Add some content
	tree.Root("Root Node")

	// We can't easily test the specific styling due to the nature of lipgloss,
	// but we can at least verify the tree renders without errors
	treeString := tree.String()
	assert.NotEmpty(t, treeString)
	assert.Contains(t, treeString, "Root Node")
}

// TestBranchNodeComplexStructure tests creating a more complex branch node structure
func TestBranchNodeComplexStructure(t *testing.T) {
	// Create a branch node
	parentNode := fancy.BranchNode("Parent", "(3)")

	// Add children
	child1 := parentNode.Child("Child 1")
	child1.Child("Grandchild 1")

	child2 := parentNode.Child("Child 2")
	child2.Child("Grandchild 2")

	// Verify the structure is correct
	treeString := parentNode.String()
	assert.Contains(t, treeString, "Parent")
	assert.Contains(t, treeString, "(3)")
	assert.Contains(t, treeString, "Child 1")
	assert.Contains(t, treeString, "Child 2")
	assert.Contains(t, treeString, "Grandchild 1")
	assert.Contains(t, treeString, "Grandchild 2")
}

// TestNewComponentTree tests the creation of a new component tree
func TestNewComponentTree(t *testing.T) {
	title := "Test Component"

	// Create a new component tree
	compTree := fancy.NewComponentTree(title)

	// Assert the tree was created
	assert.NotNil(t, compTree)

	// Check that we can get the underlying tree
	treeObj := compTree.Tree()
	assert.NotNil(t, treeObj)

	// Verify the tree has the title as root
	assert.Contains(t, treeObj.String(), title)
}

// TestAddBranch tests adding a branch to a component tree
func TestAddBranch(t *testing.T) {
	compTree := fancy.NewComponentTree("Root")
	branchText := "Branch 1"

	// Add a branch
	branch := compTree.AddBranch(branchText)

	// Assert the branch was created and is of correct type
	assert.NotNil(t, branch)

	// Verify the branch text appears in the rendered tree
	treeString := compTree.Tree().String()
	assert.Contains(t, treeString, branchText)
}

// TestAddChild tests adding a child node to the component tree
func TestAddChild(t *testing.T) {
	compTree := fancy.NewComponentTree("Root")
	childText := "Child Node"

	// Add a child
	child := compTree.AddChild(childText)

	// Assert the child was created and is of correct type
	assert.NotNil(t, child)

	// Verify the child text appears in the rendered tree
	treeString := compTree.Tree().String()
	assert.Contains(t, treeString, childText)
}

// TestEndpointTree tests creating a tree for endpoint visualization
func TestEndpointTree(t *testing.T) {
	endpointID := "test-endpoint"

	// Create an endpoint tree
	endpointTree := fancy.EndpointTree(endpointID)

	// Assert the tree was created
	assert.NotNil(t, endpointTree)

	// Verify the tree has the styled endpoint ID
	// The exact string comparison is tricky because of ANSI styling,
	// but we can check if the ID is part of the rendered string
	treeString := endpointTree.Tree().String()
	assert.Contains(t, treeString, endpointID)
}

// TestRouteTree tests creating a tree for route visualization
func TestRouteTree(t *testing.T) {
	routeInfo := "/api/v1/test"

	// Create a route tree
	routeTree := fancy.RouteTree(routeInfo)

	// Assert the tree was created
	assert.NotNil(t, routeTree)

	// Verify the tree has the styled route info
	treeString := routeTree.Tree().String()
	assert.Contains(t, treeString, routeInfo)
}

// TestTreeChaining tests the ability to chain tree operations
func TestTreeChaining(t *testing.T) {
	compTree := fancy.NewComponentTree("Root")

	// Create a more complex tree with chained operations
	branch1 := compTree.AddBranch("Branch 1")
	branch1.Child("Child 1.1")
	branch1.Child("Child 1.2")

	branch2 := compTree.AddBranch("Branch 2")
	branch2.Child("Child 2.1")

	// Verify the final tree structure contains all the elements
	treeString := compTree.Tree().String()
	assert.Contains(t, treeString, "Root")
	assert.Contains(t, treeString, "Branch 1")
	assert.Contains(t, treeString, "Child 1.1")
	assert.Contains(t, treeString, "Child 1.2")
	assert.Contains(t, treeString, "Branch 2")
	assert.Contains(t, treeString, "Child 2.1")
}

// TestStyleConsistency tests that the styles are consistently applied
func TestStyleConsistency(t *testing.T) {
	// Test endpoint and route trees use the correct styles
	endpointTree := fancy.EndpointTree("endpoint")
	routeTree := fancy.RouteTree("/path")

	// Each tree should have a different rendered string due to different styles
	assert.NotEqual(t, endpointTree.Tree().String(), routeTree.Tree().String())
}
