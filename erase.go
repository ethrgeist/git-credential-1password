package main

func eraseCommand() {
	gitInputs := ReadLines()
	// run "op delete item" command with the host value
	buildOpItemCommand("delete", itemName(gitInputs["host"])).Run()
}
