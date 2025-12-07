# [2.1.0](https://github.com/pandeptwidyaop/http-remote/compare/v2.0.0...v2.1.0) (2025-12-07)


### Features

* add security hardening and documentation improvements ([3be73ce](https://github.com/pandeptwidyaop/http-remote/commit/3be73ce725e5bd8e1273ef7f0dc457906847cd20))

# [2.0.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.12.1...v2.0.0) (2025-12-07)


### security

* require encryption key and secure admin password at startup ([4c0e40b](https://github.com/pandeptwidyaop/http-remote/commit/4c0e40be2442012486436a84b406c121c17799d6))


### BREAKING CHANGES

* config.yaml now requires:
- security.encryption_key: 64-character hex string
- admin.password: must not be "changeme"

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>

## [1.12.1](https://github.com/pandeptwidyaop/http-remote/compare/v1.12.0...v1.12.1) (2025-12-07)


### Bug Fixes

* resolve all golangci-lint issues (130 issues fixed) ([e13b001](https://github.com/pandeptwidyaop/http-remote/commit/e13b0013579acf6aa99daea0a94dc333a08a77b2))
* update golangci-lint config for v2.5.0 compatibility ([1744719](https://github.com/pandeptwidyaop/http-remote/commit/17447196037c539512f8394e27ca94bed0d8ccdd))

# [1.12.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.11.0...v1.12.0) (2025-12-06)


### Features

* add 2FA backup codes, password history, and app backup/export ([36e4586](https://github.com/pandeptwidyaop/http-remote/commit/36e45866c98c79cc9f814e6dbf9ab4448cb235a7))
* add audit logging for file management operations ([ef55270](https://github.com/pandeptwidyaop/http-remote/commit/ef552701e949583e4467542493974601f1becd11))
* add file browser with upload/download capabilities ([ce791f6](https://github.com/pandeptwidyaop/http-remote/commit/ce791f68ea6838ff8c1f939dec2f759a2e6f1502))
* add multiple terminal sessions and recording ([116dac6](https://github.com/pandeptwidyaop/http-remote/commit/116dac619f397dcf766c5bede2c2bdcd747c3935))
* add multiple terminal sessions with audit logging ([622813a](https://github.com/pandeptwidyaop/http-remote/commit/622813a93469d415cd8d393494ab6b0f32c13119))
* add password complexity, input sanitization, and request size limits ([a599ffd](https://github.com/pandeptwidyaop/http-remote/commit/a599ffd9f241558ba17fff2b9abff00ed4bd2d85))
* add persistent terminal sessions, protect default admin, and support root path_prefix ([df410a3](https://github.com/pandeptwidyaop/http-remote/commit/df410a347c922ebcb218f4b9ac0fb052d6534657))
* add user management with role-based access control ([00eee08](https://github.com/pandeptwidyaop/http-remote/commit/00eee082e9e722e69e59e7e3501fd7c8fb82b314))

# [1.11.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.10.0...v1.11.0) (2025-12-06)


### Features

* add account lockout for brute force protection ([406de47](https://github.com/pandeptwidyaop/http-remote/commit/406de4734c30b3739c4d667eb69de859329375da))
* add comprehensive audit logging to all handlers ([25a88b3](https://github.com/pandeptwidyaop/http-remote/commit/25a88b3ff9f69369075264c5826dcc5c06696bc5))
* add session binding and CSRF protection ([b6ff77f](https://github.com/pandeptwidyaop/http-remote/commit/b6ff77f2c9dfb4e1b4872c96c187d50d205d0ada))

# [1.10.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.9.0...v1.10.0) (2025-12-06)


### Features

* add fullscreen mode to terminal ([2c70fa3](https://github.com/pandeptwidyaop/http-remote/commit/2c70fa3e48a967187a15a25ae63a10e5dc81b660))
* add security hardening (WebSocket origin, TOTP encryption, headers) ([a85e137](https://github.com/pandeptwidyaop/http-remote/commit/a85e1375727138d4f02549f6d4c46997fb9e52d6))

# [1.9.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.8.0...v1.9.0) (2025-12-06)


### Features

* add version display and update notification in web UI ([5276304](https://github.com/pandeptwidyaop/http-remote/commit/527630499e6245905efe4bf2f6f108228b3dccb5))

# [1.8.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.7.0...v1.8.0) (2025-12-06)


### Features

* add terminal configuration options ([95ca81f](https://github.com/pandeptwidyaop/http-remote/commit/95ca81fc37d80be1609f8a86b75fd6b600ce89f8))

# [1.7.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.6.1...v1.7.0) (2025-12-06)


### Bug Fixes

* clean up terminal handler and fix auth tests ([9623cf2](https://github.com/pandeptwidyaop/http-remote/commit/9623cf25f882a994e2c3af5ca765a2abc7a11a74))
* terminal WebSocket path and CI/CD frontend build ([a043dbe](https://github.com/pandeptwidyaop/http-remote/commit/a043dbed4e62873e92a23ec8f3fab5d2fe3a95ce))


### Features

* add 2FA and password management with modern React UI ([4052ff8](https://github.com/pandeptwidyaop/http-remote/commit/4052ff8313006175d888559dd5d1ba29ae4998a8))
* add AES-256-GCM encryption service for TOTP secrets ([43b4565](https://github.com/pandeptwidyaop/http-remote/commit/43b4565232dfbae2e464aa749ea4237b7d235401))
* add backup codes support for 2FA recovery ([78d496f](https://github.com/pandeptwidyaop/http-remote/commit/78d496fc66926ea1edcd1c95787fe0f5d6700c14))
* add rate limiting for 2FA endpoints ([3d109e6](https://github.com/pandeptwidyaop/http-remote/commit/3d109e6d1f842db0c85659f70c78e46b83f0cb31))
* add WebSocket terminal for remote shell access ([8e4cf0c](https://github.com/pandeptwidyaop/http-remote/commit/8e4cf0c91b018776de2b7be3b0a3a6adae2821ed))

## [1.6.1](https://github.com/pandeptwidyaop/http-remote/compare/v1.6.0...v1.6.1) (2025-12-05)


### Bug Fixes

* add package comments and improve error handling ([6b01582](https://github.com/pandeptwidyaop/http-remote/commit/6b01582c247a05c5e257b48fc60dbeca1f3dbccb))
* add type assertion checks in handlers ([e672490](https://github.com/pandeptwidyaop/http-remote/commit/e672490894cb7480cabbeacc6e80ab70b5db23d7))
* handle error returns in critical paths ([91b019e](https://github.com/pandeptwidyaop/http-remote/commit/91b019ef9995bf62ff619a72d08605bdd1e3e9b4))
* improve error handling in handlers and fix rate limit headers ([88e408f](https://github.com/pandeptwidyaop/http-remote/commit/88e408fb8c33c3ae2671bb749a952f71655f087d))

# [1.6.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.5.0...v1.6.0) (2025-12-05)


### Bug Fixes

* show API deployment executions in history ([2100cf1](https://github.com/pandeptwidyaop/http-remote/commit/2100cf196d01a4815dbae74d1f53be5bf54da270))


### Features

* add CI/CD pipeline and improve test coverage from 10.8% to 19.6% ([8132983](https://github.com/pandeptwidyaop/http-remote/commit/8132983edf9456f9e96ccca952a3b9fe710a73d4))
* add comprehensive unit tests and backward-compatible migrations ([024c244](https://github.com/pandeptwidyaop/http-remote/commit/024c244c8a36384db48c6e3242efec828ca14d30))
* add new version notification banner in web UI ([d096a8a](https://github.com/pandeptwidyaop/http-remote/commit/d096a8af6e39a06629985e4960aa8d7f0390d21e))

# [1.5.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.4.0...v1.5.0) (2025-12-05)


### Features

* add streaming support for deploy API ([4cf3796](https://github.com/pandeptwidyaop/http-remote/commit/4cf37962d4d2221371874add1b0e0b722771f718))

# [1.4.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.3.0...v1.4.0) (2025-12-05)


### Bug Fixes

* remove foreign key constraint on executions.user_id ([1147de3](https://github.com/pandeptwidyaop/http-remote/commit/1147de306d7100a7f0368904824f614790be9daa))


### Features

* add Laravel-style migration tracking system ([b258272](https://github.com/pandeptwidyaop/http-remote/commit/b258272f3a88bf7af0f9b18a5c11e7f32f18723e))

# [1.3.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.2.0...v1.3.0) (2025-12-05)


### Bug Fixes

* add Audit Logs link to executions page navigation ([b68af05](https://github.com/pandeptwidyaop/http-remote/commit/b68af054a2cb7f56b6e6665076ece8a1d27ea194))
* return empty array instead of null for empty audit logs ([a33a573](https://github.com/pandeptwidyaop/http-remote/commit/a33a57383f009fcffe0395bee0c42e7184792bf7))


### Features

* add audit log viewer in web UI ([a1e9657](https://github.com/pandeptwidyaop/http-remote/commit/a1e9657e4c21815b480f12d79c99196076a8a391))
* comprehensive security improvements ([4c85132](https://github.com/pandeptwidyaop/http-remote/commit/4c8513278c132af079e9de45c43f85d1297ad527))
* integrate audit logging into handlers ([e9876b0](https://github.com/pandeptwidyaop/http-remote/commit/e9876b09bfdd5dfb2d2f8a02c531407c70d907c3))

# [1.2.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.1.0...v1.2.0) (2025-12-05)


### Features

* add self-upgrade command ([5b5689c](https://github.com/pandeptwidyaop/http-remote/commit/5b5689c04a8a19aa03830e84b1e4edeaded9148c))

# [1.1.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.0.0...v1.1.0) (2025-12-05)


### Features

* add edit command functionality and fix long command display ([f83d568](https://github.com/pandeptwidyaop/http-remote/commit/f83d56801233af37c78fd412c6da98ec6bec1060))

# 1.0.0 (2025-12-05)


### Bug Fixes

* simplify build to linux-amd64 only ([a4abf55](https://github.com/pandeptwidyaop/http-remote/commit/a4abf5533598a9e6a6f04eb6f5fa8119be32733d))


### Features

* initial release of HTTP Remote ([166e96e](https://github.com/pandeptwidyaop/http-remote/commit/166e96e1e53245353b5f22068954ef285ca05b8b))
