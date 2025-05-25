# [1.2.0](https://github.com/Mort4lis/memdb/compare/v1.1.0...v1.2.0) (2025-05-25)


### Bug Fixes

* add segment directory integration and save functionality ([cec0f35](https://github.com/Mort4lis/memdb/commit/cec0f3590eb0a9a88c6933e6e310bf920769f879))
* fix and improve segment handling logic in WAL filesystem ([80eead6](https://github.com/Mort4lis/memdb/commit/80eead69f8d902cfd23107f29068b5f824292295))


### Features

* add sharded in-memory engine with configurable partitions ([76d99d8](https://github.com/Mort4lis/memdb/commit/76d99d831061efd8f265c5299265e416fd36620f))
* implement master-slave replication ([4a40529](https://github.com/Mort4lis/memdb/commit/4a40529bda789ccfca19b50514aa31624724cd8f))

# [1.1.0](https://github.com/Mort4lis/memdb/compare/v1.0.0...v1.1.0) (2025-05-03)


### Bug Fixes

* change WAL implementation to improve robustness and clarity ([7e49a7d](https://github.com/Mort4lis/memdb/commit/7e49a7d3fc62adbc8c36dbe4546f44e556a1584e))
* restore WAL ([8787776](https://github.com/Mort4lis/memdb/commit/8787776ee00a3f45b491c5390411e80b6c2883d4))


### Features

* add WAL support to storage layer ([7e3401b](https://github.com/Mort4lis/memdb/commit/7e3401b258cad311122038104935b6d3f77977b9))

# 1.0.0 (2025-02-15)


### Bug Fixes

* config ([72af3e2](https://github.com/Mort4lis/memdb/commit/72af3e2b0b5e7ff37ef5b3963312edb8d63d4581))
* fix semantic release actions ([25f05d6](https://github.com/Mort4lis/memdb/commit/25f05d6cf33da70b9c90e62585225ab9a76e9d90))
* tcp server shutdown reworked ([d759078](https://github.com/Mort4lis/memdb/commit/d7590786fc29a7f5d495bdf8c42ec9313ec9ece4))
* tcp_client ([4b8ccd7](https://github.com/Mort4lis/memdb/commit/4b8ccd799833d96f4293bb655dc71ba2cb87ac46))
* wait group bug ([77ee782](https://github.com/Mort4lis/memdb/commit/77ee782bfeb20bfadfa431624049ca3df742ba00))


### Features

* configuring tcp server ([13d34cd](https://github.com/Mort4lis/memdb/commit/13d34cd07974518ae75b97c687348f510e331370))
* implement basic tcp server ([d470853](https://github.com/Mort4lis/memdb/commit/d4708535f1d329e7b23ff51ffb751aabfa7d429b))
* implement command line interface for memdb ([f243cae](https://github.com/Mort4lis/memdb/commit/f243cae560f944bc0918606126228b1713340d70))
* implement tcp client ([5ac16ca](https://github.com/Mort4lis/memdb/commit/5ac16caa064ae8b3c3411432c1a1ff3db679a196))
* initialize project layout ([3bbdd77](https://github.com/Mort4lis/memdb/commit/3bbdd77d3ab303460a460472e715b54c9d513d5d))
