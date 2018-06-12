# TODO / Goals / Questions

This document contains open questions and goals regarding the development of this project. You can think of it like 'stuff to do next'. We hope this text will evolve into an FAQ, where multiple thinkings regarding the architecture of this project will be detailed. Random order

- Pod autoscaling
- Cluster autoscaling
- Custom scheduler
- What happens if a Pod dies?
- Health checks?
- How can we prevent explicit Pod deletion? We should mark it as 'MarkedForDeletion'
- Can the user arbitrarily delete DedicatedGameServers that are members of a DedicatedGameServerCollection?
- Should we use hostPort? hostNetwork?
- Multiple ports per game server?
- setSessionsURL?
- Image updates - how?