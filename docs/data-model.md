# data model

## hierarchy 
        
organization
  └── project (1..n)
        ├── name, description, created_at
        ├── members[] (rbac)
        └── environment (production, staging, dev, preview)
          └── cluster (1..n per project)
                ├── name
                ├── template: "single"|"ha"|"read-replicas"|"custom"
                ├── desired_state: JSONB
                ├── observed_state: JSONB
                ├── host_id → Host
                │
                ├── nodes[]
                │     ├── role: primary | replica
                │     ├── host_id → host (can differ per node)
                │     └── databases[]
                │           ├── name, owner
                │           └── extensions[]
                │                 ├── name, desired_version
                │                 └── node_states[]
                │
                └── proxies[]
                      ├── type, scope, pool_mode
                      └── config: JSONB

host (independent entity, belongs to org not project)
  ├── token, hostname, capabilities
  └── agent (1 per host)

### activity log

event {
  project_id
  environment_id
  actor: "user:winit" | "agent:host-01" | "autopilot"
  action: "failover" | "backup_completed" | "extension_enabled" | ...
  before_state: JSONB
  after_state: JSONB
  timestamp
}

### extension reconcilation

compatibility_issue {
  type: "version_drift" | "presence_drift" | "availability_drift" | "conflict"
  severity: "warning" | "error"      - error blocks apply, warning prompts confirm
  affected_nodes: node_id[]
  affected_databases: database_id[]
  message: string                    - human readable, shown in canvas
  auto_resolvable: boolean           - can wiant fix this without user input?
}
