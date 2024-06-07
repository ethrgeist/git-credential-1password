package main

func eraseCommand() {
	gitInputs := ReadLines()

	itemId := findItemId(gitInputs["protocol"], gitInputs["host"])
	if itemId != nil {
		// run "op delete item" command with the found item id
		buildOpItemCommand("delete", *itemId).Run()
	}
}
