Demo 1
* Give it a link  https://www.anthropic.com/engineering/writing-tools-for-agents
* 





```bash
curl -X POST https://unreciprocally-superscholarly-fabian.ngrok-free.dev/api/v1/articles \
    -H "Content-Type: application/json" \
    -d '{
      "url": "https://www.anthropic.com/engineering/writing-tools-for-agents",
      "format": "audio",
      "length": "m"
    }'
```