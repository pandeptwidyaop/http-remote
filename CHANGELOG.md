# [2.0.0](https://github.com/pandeptwidyaop/http-remote/compare/v1.6.0...v2.0.0) (2025-12-06)


### Features

* **2FA/TOTP Authentication**: Add two-factor authentication with TOTP support using authenticator apps ([2c332c3](https://github.com/pandeptwidyaop/http-remote/commit/2c332c3), [4052ff8](https://github.com/pandeptwidyaop/http-remote/commit/4052ff8))
  - QR code generation for easy setup
  - TOTP verification on login
  - Encrypted secret storage using AES-256-GCM ([43b4565](https://github.com/pandeptwidyaop/http-remote/commit/43b4565))
* **Backup Codes**: Add encrypted recovery codes for 2FA account recovery ([78d496f](https://github.com/pandeptwidyaop/http-remote/commit/78d496f))
* **Remote Terminal**: WebSocket-based interactive shell access with PTY support ([8e4cf0c](https://github.com/pandeptwidyaop/http-remote/commit/8e4cf0c))
  - Real-time terminal emulation using xterm.js
  - Full shell access for authenticated users
  - Bi-directional communication via WebSocket
* **Modern React SPA UI**: Migrated from vanilla JS to React 18 with TypeScript and Tailwind CSS ([2c332c3](https://github.com/pandeptwidyaop/http-remote/commit/2c332c3))
  - Built with Vite for fast development
  - Embedded in binary using Go embed ([5c995f2](https://github.com/pandeptwidyaop/http-remote/commit/5c995f2))
* **Enhanced Rate Limiting**: Add rate limiting for 2FA endpoints (10 req/min) ([3d109e6](https://github.com/pandeptwidyaop/http-remote/commit/3d109e6))
* **Password Management**: Add change password functionality in Settings
* **AES-256-GCM Encryption**: Implement encryption service for sensitive data ([43b4565](https://github.com/pandeptwidyaop/http-remote/commit/43b4565))


### Documentation

* Update README with v2 features and build instructions ([08d0e78](https://github.com/pandeptwidyaop/http-remote/commit/08d0e78))
  - Add Node.js requirement for frontend build
  - Document 2FA setup process
  - Document remote terminal usage


### Bug Fixes

* Clean up terminal handler and fix auth tests ([9623cf2](https://github.com/pandeptwidyaop/http-remote/commit/9623cf2))
  - Remove unused terminal session tracking
  - Fix test database schema for 2FA columns


### Breaking Changes

* **Frontend**: Migrated from vanilla JS to React SPA - custom modifications to old frontend will need migration
* **Database Schema**: Added new columns for 2FA support (totp_secret, totp_enabled, backup_codes) - migrations run automatically


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
