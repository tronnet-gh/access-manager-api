# access-manager-api - Simple Multi-Realm Access Management Endpoint
access-manager-api provides functionality for the API by implementing a single set of endpoints for all ProxmoxAAS user management. It supports pve and ldap type users.

# Permission System

Note that this only reflects permissions granted or not granted through access-manager-api. PVE permissions could allow a Realm Admin User to add users/groups to PVE realm, but will not be explicitly supported by PAAS api. Some permissions are granted by the inclusion of the role on an ACL list.

Native Proxmox Client User

| Operation | Object Realm | Required Permission | Permission Backend |
|---|---|---|---|
| Manage Instances | PVE | PAAS Quota & PVE Pool Membership Check | PAAS |
| Power Management | PVE | PVE VM.PowerMgmt | PVE |
| Backup Management | PVE | PAAS Quota & PVE Pool Membership Check | PAAS |
| Console | PVE | PVE VM.Console | PVE |
| Change Password | PVE | N/A (all PVE users can change their password) | PVE |

External Realm Client User

| Operation | Object Realm | Required Permission | Permission Backend |
|---|---|---|---|
| Manage Instances | PVE | PAAS Quota & PVE Pool Membership Check | PAAS |
| Power Management | PVE | PVE VM.PowerMgmt | PVE |
| Backup Management | PVE | PAAS Quota & PVE Pool Membership Check | PAAS |
| Console | PVE | PVE VM.Console | PVE |
| Change Password | Realm | Relevant Realm Permission (ie ldap olcAccess to userPassword attr) | Realm |

Native Proxmox Admin User
| Operation | Object Realm | Required Permission | Permission Backend |
|---|---|---|---|
| Manage Users | PVE | PVE User.Modify & PVE Realm.AllocateUser | PVE |
| Manage Groups | PVE | PVE Group.Allocate | PVE |
| Manage Users | Realm | No Permission | N/A |
| Manage Groups | Realm | No Permission | N/A |
| Manage Pools | PVE | PVE Pool.Allocate | PVE |
| Assign Groups to Pools | PVE | PVE Pool.Allocate & PVE Group.Allocate | PVE |
| Assign Users to Groups | PVE | PVE Group.Allocate & PVE User.Modify | PVE |

External Realm Admin User
| Manage Users | PVE | No Permission | N/A |
| Manage Groups | PVE | No Permission | N/A |
| Manage Users | Realm | Realm Modify User | N/A |
| Manage Groups | Realm | Realm Modify Group | N/A |
| Manage Pools | PVE | PVE Pool.Allocate | PVE |
| Assign Groups to Pools | PVE | PVE Pool.Allocate & PVE Group.Allocate | PVE |
| Assign Users to Groups | PVE | PVE Group.Allocate & PVE User.Modify | PVE |

## PAAS-Client Role

Used to give members of a group the ability to use ProxmoxAAS API services on a pool. When a group is given the PAAS-Client role on a pool, members of the group which have confirugrations in the database will be able to create, delete, and manage VMs and CTs in that pool.

The role requires:
- Pool.Audit
- Datastore.Audit
- VM.Audit
- VM.Console
- VM.PowerMgmt

ProxmoxAAS effectively allows members of groups with Pool.Audit, VM.Audit, and Datastore.Audit to modify VM/CT configuration, and Datastore (disks) in a controlled way. Single user ACL entries are not supported.

## PAAS-Admin Role

Used to give members of a group the ability to modify ProxmoxAAS pool quotas and assignments. When a group is given the PAAS-Client role on a pool, members of that group may edit the database values of that pool, but modifying values stored in Proxmox or LDAP (ie assigning ACLs or adding users to groups) fall back to those permission systems instead. It is intended to be about as powerful as the built-in Administrator role, but can be less if desired.

The role requires:
- (recommended) User.Modify & Realm.AllocateUser
- (recommended) Group.Allocate
- (recommended) Pool.Allocate
- (recommended) Permissions.Modify

## Installation

### Prerequisites
- [ProxmoxAAS-API](https://git.tronnet.net/tronnet/ProxmoxAAS-API)
- Proxmox VE Cluster (v7.0+)

### Installation - API
1. Download `access-manager-api` binary and `template.config.json` file from releases
2. Rename `template.config.json` to `config.json` and modify the following values:
    1. `listenPort`: port for access-manager-api to bind and listen on
    2. `url`: the URI to the Proxmox API, ie `https://pve.domain.net/api2/json`
    3. `token`: the `proxmoxaas-api` API token user(name), authentication realm (pam), token id, and token secrey key (uuid)
    4. `paas-client-role`: name of the Role which represents PAAS users, ie `PAAS-Client`