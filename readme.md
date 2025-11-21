# An agent to play the hive arena 2025
https://github.com/hivehelsinki/hive-arena-2025
## Quick start

Clone above repo into folder `arena`, and this agent into a sibling folder `agent`.

go to agent folder, run 
```
go mod init https://github.com/patsastus/hive_arena_2025.git
go mod tidy
```

In the folder containing both arena and agent, run
```
go work init
go work use ./arena
go work use ./agent

```

For testing: in arena folder:
```
go run ./server
```

in parent folder:
```
cp agent/dev_match match.go
go work init runner
go work use .
go run match.go
```


Run `go run . <host> <gameid> <name>` in the agent's directory to join the game `gameid` on the arena server running at `host`. `name` is a free string you can use to name your agent or team in the game logs.

For instance: `go run . localhost:8000 bright-crimson-elephant-0 SuperTeam`





