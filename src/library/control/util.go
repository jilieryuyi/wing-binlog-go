package control

func hasCmd(cmd int) bool {
	return cmd == CMD_ERROR||
		cmd == CMD_TICK ||
		cmd == CMD_STOP||
		cmd == CMD_RELOAD||
		cmd == CMD_SHOW_MEMBERS
}