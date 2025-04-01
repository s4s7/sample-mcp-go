## Sample-mcp-go
This project is a sample project to demonstrate how to use the mcp-go library

### historical event

```
❯ curl http://localhost:8000/mcp/sse                                    
event: endpoint
data: http://localhost:8000/mcp?sessionId=6b385960-ef48-4b4c-9503-66e37d64c907
```

```
❯ curl -X POST "http://localhost:8000/mcp?sessionId=6b385960-ef48-4b4c-9503-66e37d64c907" \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc": "2.0",
       "method": "tools/call",
       "params": {
         "name": "historical_events",
         "arguments": {
           "date": "2023-09-15"
         }
       },
       "id": 2
     }'
{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"On September 15 2023:\n1. 1972 - A United States Air Force fighter jet shoots down a North Vietnamese MiG-21 over North Vietnam, marking the last aerial victory of the Vietnam War for the U.S.\n2. 2023 - A major strike by the Writers Guild of America (WGA) against the Alliance of Motion Picture and Television Producers (AMPTP) continued, significantly impacting television and film production, entering its 148th day."}]}}
```