# An agent to play the hive arena 2025
https://github.com/hivehelsinki/hive-arena-2025
## Quick start

Run `go run . <host> <gameid> <name>` in the agent's directory to join the game `gameid` on the arena server running at `host`. `name` is a free string you can use to name your agent or team in the game logs.

For instance: `go run . localhost:8000 bright-crimson-elephant-0 SuperTeam`

Run `lua test.lua <host>`.
