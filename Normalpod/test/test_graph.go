package main

import (
	"fmt"
	"github.com/meixiezichuan/broadcast-gossip/common"
)

func main() {
	g := common.NewGraph()
	//g.AddEdge("p", "n1")
	//g.AddEdge("p", "s")
	//g.AddEdge("p", "n3")
	//
	//g.AddEdge("n1", "s")
	//g.AddEdge("n1", "n2")
	//
	//g.AddEdge("s", "n2")
	//g.AddEdge("s", "l")
	//g.AddEdge("s", "n1")
	//g.AddEdge("s", "n3")
	//
	//g.AddEdge("n3", "n4")
	//
	//g.AddEdge("n4", "n5")
	//
	//g.AddEdge("l", "s")
	//g.AddEdge("l", "n5")
	//g.AddEdge("l", "n2")

	//g.AddEdge("1", "2")
	//g.AddEdge("1", "3")
	//g.AddEdge("2", "4")
	//g.AddEdge("2", "5")
	//g.AddEdge("3", "6")
	//g.AddEdge("3", "7")
	//g.AddEdge("6", "8")

	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("B", "D")
	g.AddEdge("B", "E")
	g.AddEdge("C", "F")
	g.AddEdge("D", "E")
	g.AddEdge("E", "F")

	root := "A"
	g.Sotred()

	fmt.Println("Graph: ")
	g.Display()

	mlst := g.FindMaxLeafTree(root)
	fmt.Println("Mlst: ")
	mlst.Display()

	//mlst2, leafcount, leaves := g.MaxLeafSpanningTree("p")
	//fmt.Println("Mlst2: ")
	//mlst2.Display()
	//fmt.Println("leafcount: ", leafcount, "leaves: ", leaves)

	leafcount, mlst2, leaves := g.MLST2DFS(root)
	fmt.Println("Mlst2: ", mlst2)
	fmt.Println("leafcount: ", leafcount, "leaves: ", leaves)

	leafcount3, mlst3, leaves3 := g.MLSTBFS(root)
	fmt.Println("Mlst3: ", mlst3)
	fmt.Println("leafcount3: ", leafcount3, "leaves3: ", leaves3)

	leafcount4, leaves4 := g.MLST4(root)
	fmt.Println("leafcount4: ", leafcount4, "leaves4: ", leaves4)

	leafcount5, leaves5 := g.MLST5(root)
	fmt.Println("leafcount5: ", leafcount5, "leaves5: ", leaves5)

	leafcount6 := g.MLST6(root)
	fmt.Println("leafcount6: ", leafcount6)
	//mdst := g.BuildMDSTree("p")
	//fmt.Println("MDS: ")
	//mdst.Display()

	tree6 := g.ConnectRootToMDS(root)
	fmt.Println("tree6: ")
	tree6.Display()

	tree10, c := g.MLST10(root)
	fmt.Println("tree10: ")
	tree10.Display()
	fmt.Println("leaves count10: ", c)

	//
	//bst := g.BST("p")
	//fmt.Println("BST: ")
	//bst.Display()
}
