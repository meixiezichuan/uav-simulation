package common

import (
	"fmt"
	"sort"
)

// MaxLeafSpanningTree 计算最大叶子生成树
func (g *Graph) MLST4(root string) (int, map[string][]string) {
	// DPState 定义状态
	type DPState struct {
		maxLeaves int
	}
	dp := make(map[string]DPState)
	visited := make(map[string]bool)
	tree := make(map[string][]string)

	var dfs func(node string) int
	dfs = func(node string) int {
		visited[node] = true
		state := DPState{0}

		childrenLeaves := 0
		isLeaf := true // 假设当前节点是叶子节点

		// 使用 sync.Map 获取邻居节点
		if neighbors, ok := g.adjList.Load(node); ok {
			for _, neighbor := range neighbors.([]string) {
				if visited[neighbor] {
					continue
				}
				isLeaf = false
				childLeaves := dfs(neighbor) // 递归获取子节点的叶子数量
				childrenLeaves += childLeaves
				tree[node] = append(tree[node], neighbor) // 记录树结构
			}
		}

		if isLeaf {
			state.maxLeaves = 1 // 叶子节点自身计数
		} else {
			state.maxLeaves = childrenLeaves // 不是叶子节点，累加子节点的叶子数量
		}

		dp[node] = state // 保存当前节点状态
		return state.maxLeaves
	}

	dfs(root) // 从根节点开始 DFS

	// 查找最大叶子数量
	maxLeaves := 0
	for _, state := range dp {
		if state.maxLeaves > maxLeaves {
			maxLeaves = state.maxLeaves
		}
	}

	return maxLeaves, tree
}

// 贪心算法
func (g *Graph) MLST5(root string) (int, []string) {
	visited := make(map[string]bool)
	var maxLeaves int
	var leafNodes []string

	var dfs func(node string) int
	dfs = func(node string) int {
		visited[node] = true
		isLeaf := true
		children := 0

		// 使用 sync.Map 获取邻居节点
		if neighbors, ok := g.adjList.Load(node); ok {
			for _, neighbor := range neighbors.([]string) {
				if !visited[neighbor] {
					isLeaf = false
					children++
					dfs(neighbor)
				}
			}
		}

		if children == 0 {
			isLeaf = true
		}
		if isLeaf {
			leafNodes = append(leafNodes, node) // 记录叶子节点
			return 1
		}
		return children
	}

	dfs(root)
	maxLeaves = len(leafNodes)
	return maxLeaves, leafNodes
}

func (g *Graph) MLST6(root string) int {
	visited := make(map[string]bool)
	leafCount := 0

	var dfs func(node string) int
	dfs = func(node string) int {
		visited[node] = true
		isLeaf := true
		children := 0

		// 使用 sync.Map 获取邻居节点
		if neighbors, ok := g.adjList.Load(node); ok {
			for _, neighbor := range neighbors.([]string) {
				if !visited[neighbor] {
					isLeaf = false
					children++
					dfs(neighbor)
				}
			}
		}

		if isLeaf {
			leafCount++
		}

		return children
	}

	dfs(root)
	return leafCount
}

// Function to connect root to MDS nodes and minimize tree size
func (g *Graph) ConnectRootToMDS(root string) *Graph {
	mds := g.MinDominatingSetFromRoot(root)
	fmt.Println("mds: ", mds)
	tree := NewGraph()
	tree.root = root
	connected := map[string]bool{root: true}

	// 使用 sync.Map 获取邻居节点
	if neighbors, ok := g.adjList.Load(root); ok {
		for _, node := range neighbors.([]string) {
			connected[node] = true
			tree.AddEdge(root, node)
		}
	}

	// Add the root to the tree
	for _, node := range mds {
		if node != root {

			// 如果节点是根节点的邻居，直接连接，
			// 否则节点是根的孙子节点，因为图是三度深
			if !connected[node] {
				// 否则，找到最佳父节点来连接节点
				_ = findBestParent(g, node, connected, mds, tree)
			}
		}
	}

	return tree
}

// Helper function to find the best parent for a given node
func findBestParent(g *Graph, node string, connected map[string]bool, mds []string, tree *Graph) string {
	var bestn string
	neighbors := []string{}
	// 使用 sync.Map 获取邻居节点
	if nbs, ok := g.adjList.Load(node); ok {
		neighbors = nbs.([]string)
	}
	sort.Strings(neighbors)

	bestn, _ = g.findMaxMdsNode(neighbors, mds)

	// node 是根的孙子节点
	if connected[bestn] {
		tree.AddEdge(bestn, node)
		return bestn
	}

	// node 是根的曾孙节点
	nns := []string{}
	// 使用 sync.Map 获取邻居节点
	if nnb, ok := g.adjList.Load(bestn); ok {
		nns = nnb.([]string)
	}
	sort.Strings(nns)
	bestnn, _ := g.findMaxMdsNode(nns, mds)
	connected[bestn] = true
	tree.AddEdge(bestn, bestnn)
	connected[node] = true
	tree.AddEdge(bestn, node)
	return bestn
}

func (g *Graph) findMaxMdsNode(nodes []string, mds []string) (string, int) {
	neighbors := nodes
	sort.Strings(neighbors)

	bestn := neighbors[0]
	mxmdsc := 0
	for _, n := range neighbors {
		mdsc := 0
		// 使用 sync.Map 获取邻居节点
		if adj, ok := g.adjList.Load(n); ok {
			for _, nn := range adj.([]string) {
				if Contains(mds, nn) {
					mdsc++
				}
			}
		}
		if mdsc > mxmdsc {
			mxmdsc = mdsc
			bestn = n
		}
	}
	return bestn, mxmdsc
}

func (g *Graph) MLST9(root string) (*Graph, []string) {
	visited := make(map[string]bool)
	tree := make(map[string][]string)
	var leaves []string
	mlstree := NewGraph()
	mlstree.root = root

	visited[root] = true
	var dfs func(node string, parent string)
	dfs = func(node string, parent string) {

		children := []string{}
		// 使用 sync.Map 获取邻居节点
		if neighbors, ok := g.adjList.Load(node); ok {
			for _, neighbor := range neighbors.([]string) {
				if !visited[neighbor] {
					children = append(children, neighbor)
				}
			}
		}

		if len(children) == 0 {
			leaves = append(leaves, node) // 如果没有未访问的子节点，标记为叶子节点
		}

		// 记录子节点信息并按照能产生最多叶子的方式选择连接
		childCounts := make([]int, len(children))
		for i, child := range children {
			dfs(child, node)
			childCounts[i] = len(tree[child]) // 记录每个子节点的叶子数量
		}

		// 选择能产生最多叶子的节点进行连接
		maxIndex := -1
		maxLeaves := -1
		for i, count := range childCounts {
			if count > maxLeaves {
				maxLeaves = count
				maxIndex = i
			}
		}

		if maxIndex != -1 {
			child := children[maxIndex]
			visited[child] = true
			mlstree.AddEdge(node, child)
			tree[node] = append(tree[node], children[maxIndex]) // 连接到最多叶子的子节点
		}
	}

	dfs(root, "")

	return mlstree, leaves
}

func (g *Graph) MLST10(root string) (*Graph, []string) {
	g.Sotred()
	mlstree := NewGraph()
	mlstree.root = root
	var leaves []string
	connected := map[string]bool{root: true}

	// 使用 sync.Map 获取 root 节点的邻居
	if neighbors, ok := g.adjList.Load(root); ok {
		for _, node := range neighbors.([]string) {
			mlstree.AddEdge(root, node)
			connected[node] = true
		}
	}

	count := 0
	g.adjList.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if len(connected) == count {
		return mlstree, leaves
	}

	// since graph only include 1-hop and 2-hop nodes, thus the most height of tree is 3
	// child of root
	maxUnconnected := -1
	var nodeSelected string
	var parent string
	var nodel []string

	// 遍历根节点的邻居节点
	if neighbors, ok := g.adjList.Load(root); ok {
		for _, node := range neighbors.([]string) {
			// 遍历每个邻居的邻居
			if nbs, ok := g.adjList.Load(node); ok {
				for _, nn := range nbs.([]string) {
					if !connected[nn] {
						unconnected, l := g.findChildUnconnected(nn, connected)
						if maxUnconnected < unconnected {
							maxUnconnected = unconnected
							nodeSelected = nn
							parent = node
							nodel = l
						}
					}
				}
			}
		}
	}

	// 连接节点
	mlstree.AddEdge(root, parent)
	connected[parent] = true
	mlstree.AddEdge(parent, nodeSelected)
	connected[nodeSelected] = true
	for _, l := range nodel {
		mlstree.AddEdge(nodeSelected, l)
		connected[l] = true
	}

	// 处理其他未连接的节点
	nodes := []string{}
	// 获取所有节点的键
	g.adjList.Range(func(key, _ interface{}) bool {
		nodes = append(nodes, key.(string))
		return true
	})
	sortedNodes := g.GetSortedNodes(nodes)
	for _, node := range sortedNodes {
		if !connected[node] {
			_, neigh := g.findMaxConnectedNeighbor(node, connected)
			mlstree.AddEdge(neigh, node)
			connected[neigh] = true
		}
	}
	// 获取叶子节点
	leaves = mlstree.getLeaves()
	return mlstree, leaves
}

// findChildUnconnected 查找某个节点的未连接子节点
func (g *Graph) findChildUnconnected(node string, connected map[string]bool) (int, []string) {
	unconnected := 0
	var uns []string

	// 使用 sync.Map 获取邻居节点
	if neighbors, ok := g.adjList.Load(node); ok {
		for _, n := range neighbors.([]string) {
			if !connected[n] {
				unconnected++
				uns = append(uns, n)
			}
		}
	}
	return unconnected, uns
}

// findMaxConnectedNeighbor 查找一个节点最连接的邻居
func (g *Graph) findMaxConnectedNeighbor(node string, connected map[string]bool) (int, string) {
	maxnn := -1
	var maxNeighbor string

	// 使用 sync.Map 获取邻居节点
	if neighbors, ok := g.adjList.Load(node); ok {
		for _, n := range neighbors.([]string) {
			connectd := 0
			// 遍历邻居的邻居，计算已连接的邻居数量
			if nbs, ok := g.adjList.Load(n); ok {
				for _, nn := range nbs.([]string) {
					if connected[nn] {
						connectd++
					}
				}
			}
			if maxnn < connectd {
				maxnn = connectd
				maxNeighbor = n
			}
		}
	}
	return maxnn, maxNeighbor
}

// getLeaves 获取树的叶子节点
func (g *Graph) getLeaves() []string {
	var leaves []string

	// 遍历所有节点并查找叶子节点
	g.adjList.Range(func(key, value interface{}) bool {
		node := key.(string)
		neighbors := value.([]string)
		if node != "root" && len(neighbors) == 1 {
			leaves = append(leaves, node)
		}
		return true
	})

	return leaves
}
