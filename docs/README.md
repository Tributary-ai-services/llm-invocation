# LLM Invocation Package - Documentation

## Design Documents

### User-Controlled API Key Management
- **[Design Document](./user-key-management-design.md)** - Complete technical design for BYOK (Bring Your Own Keys)
- **[Implementation Roadmap](./user-key-roadmap.md)** - 5-phase roadmap from simple to enterprise-grade

## Quick Reference

### Phase Overview
| Phase | Name | Duration | Complexity | Key Features |
|-------|------|----------|------------|--------------|
| 1 | Request Embedding | 1 week | ⭐⭐ | AES-256-GCM encryption, no dependencies |
| 2 | Session Management | 1 week | ⭐⭐⭐ | Redis caching, performance optimization |
| 3 | JWT Tokens | 1 week | ⭐⭐⭐⭐ | Standards-compliant, time-limited access |
| 4 | KMS Integration | 1 week | ⭐⭐⭐⭐⭐ | AWS/Azure/Vault, HSM backing |
| 5 | Enterprise | 2+ weeks | ⭐⭐⭐⭐⭐ | Multi-tenant, RBAC, compliance |

### Security Progression
- **Phase 1**: End-to-end encryption, no backend storage
- **Phase 2**: Session-based caching with encryption
- **Phase 3**: JWT-based access with revocation
- **Phase 4**: HSM-backed encryption with audit trails
- **Phase 5**: Enterprise compliance and multi-tenancy

### Implementation Strategy
Start with **Phase 1** (Request Embedding) as it provides:
- ✅ Immediate user control over API keys
- ✅ Strong security with no infrastructure dependencies  
- ✅ Simple client integration
- ✅ Foundation for all subsequent phases

Each phase builds upon the previous, allowing incremental rollout and testing.