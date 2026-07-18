- [ ] Make an interactive cli for setup
- [ ] Make a cli for allowing config through flags


Middleware responsibilities

API key authentication
Rate limiting
Usage quotas (daily/monthly/token limits)
Model allow/deny checks
Model alias mapping (gpt-4 → openai/gpt-4.1)
Request logging
Rejecting invalid requests



Proxy responsibilities

Read the validated request
Determine the upstream URL/provider
Add provider authentication
Rewrite the request if needed
Forward it
Stream the response back
Handle provider errors