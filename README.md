syimiql is a toy SQL database written for educational purposes. The goal is to build a toy version of [Multigres](https://github.com/multigres/multigres) so that I can learn about what it takes to build something like Multigres.

syimiql features the following:
- Postgres-like design with MVCC and WAL
- Postgres-like wire protocol, it should work with pgBouncer
- A small subset of Postgres-like SQL
- HA built on top of [Flexible Paxos](https://fpaxos.github.io/)-based consensus protocol
- Sharding
- CDC to Iceberg using [Materializer](https://multigres.com/docs#materializer)
