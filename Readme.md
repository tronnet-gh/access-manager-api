# access-manager-api - Simple Multi-Realm Access Management Endpoint
access-manager-api provides functionality for the API by implementing a single set of endpoints for all ProxmoxAAS user management. It supports pve and ldap type users.

## Installation

### Prerequisites
- [ProxmoxAAS-API](https://git.tronnet.net/tronnet/ProxmoxAAS-API)
- Proxmox VE Cluster (v7.0+)

### Installation - API
1. Download `access-manager-api` binary and `template.config.json` file from releases
2. Rename `template.config.json` to `config.json` and modify the following values:
    1. `listenPort`: port for access-manager-api to bind and listen on
    2. In `pve`:
        - `url`: the URI to the Proxmox API, ie `https://pve.domain.net/api2/json`
        - `token`: the `proxmoxaas-api` user(name), authentication realm (pam), token id, and token secrey key (uuid)
        - `paas-client-role`: name of the Role which represents PAAS users, ie `PAAS-Client`