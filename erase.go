package main

func eraseCommand() {
	gitInputs := ReadLines()

	itemId := findItemId(gitInputs["host"])
	if itemId != nil {
		// run "op delete item" command with the found item id
		buildOpItemCommand("delete", *itemId).Run()
	}
}
