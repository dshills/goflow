# Security Policy

## Supported Versions

GoFlow is currently in active development (pre-1.0 release). Security updates are provided for:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| 1.0.0-alpha | :white_check_mark: |
| < 1.0   | :x:                |

## Security Model

GoFlow implements defense-in-depth security principles:

### Credential Protection
- **System Keyring**: Credentials stored in OS-native keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- **Never in Workflows**: Workflow YAML files never contain secrets
- **Memory Protection**: Sensitive data zeroed after use

### Input Validation
- **Path Validation**: 6-layer defense against directory traversal ([CLAUDE.md](CLAUDE.md#security-implementation-notes))
- **Expression Sandboxing**: All user expressions run in restricted sandbox
- **YAML Schema Validation**: Strict workflow schema enforcement
- **Type Safety**: Strong typing with runtime validation

### Execution Safety
- **No Arbitrary Code**: No `eval()` or code execution from workflows
- **Timeout Protection**: Configurable execution timeouts prevent runaway processes
- **Resource Limits**: Memory and connection limits enforced
- **Audit Logging**: Complete execution history with security events

### MCP Protocol Security
- **Connection Validation**: Server authenticity verification
- **Schema Enforcement**: MCP tool schemas validated
- **Transport Security**: Support for secure transports (HTTPS, authenticated SSE)
- **Sandboxed Tools**: MCP tool execution isolated from GoFlow core

## Reporting a Vulnerability

**Please do NOT report security vulnerabilities through public GitHub issues.**

### How to Report

1. **Email**: Send details to **security@[yourdomainhere]** (or create a private security advisory)
2. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Affected versions
   - Potential impact
   - Suggested fix (if available)

### Response Timeline

- **24 hours**: Acknowledgment of report
- **7 days**: Initial assessment and triage
- **30 days**: Target for patch development
- **Coordinated disclosure**: After patch is available

### What to Expect

1. **Acknowledgment**: We'll confirm receipt and assign a tracking ID
2. **Assessment**: We'll evaluate severity using CVSS v3.1
3. **Development**: Patch developed and tested
4. **Disclosure**: Coordinated release with credit to reporter
5. **Advisory**: Security advisory published with details

### Severity Levels

| Level | Response Time | Description |
|-------|---------------|-------------|
| **Critical** | 24-48 hours | RCE, credential exposure, data breach |
| **High** | 7 days | Privilege escalation, DoS, path traversal |
| **Medium** | 30 days | Information disclosure, input validation bypass |
| **Low** | 60 days | Minor issues with limited impact |

## Security Best Practices

### For Users

1. **Keep Updated**: Use the latest GoFlow version
2. **Verify Workflows**: Review workflows before execution
3. **Limit MCP Servers**: Only register trusted MCP servers
4. **Monitor Executions**: Review execution logs regularly
5. **Use Path Validation**: Ensure filesystem operations stay in bounds
6. **Secure Credentials**: Use keyring for all sensitive data

### For Workflow Authors

1. **Validate Inputs**: Always validate workflow input variables
2. **Restrict Paths**: Use workflow-relative paths, not absolute paths
3. **Handle Errors**: Implement proper error handling and retries
4. **Document Risks**: Note any security considerations in workflow metadata
5. **Test Expressions**: Validate all transformation expressions
6. **Limit Scope**: Request minimum necessary MCP tool permissions

### For MCP Server Developers

1. **Schema Validation**: Enforce strict input schemas
2. **Error Messages**: Don't leak sensitive info in errors
3. **Rate Limiting**: Implement rate limits for expensive operations
4. **Audit Logs**: Log security-relevant events
5. **Least Privilege**: Request minimum necessary permissions

## Known Security Considerations

### Current Limitations

1. **Local Execution Only**: GoFlow runs with user's permissions
2. **MCP Server Trust**: Users must trust registered MCP servers
3. **Expression Language**: Limited subset for safety (no file/network access)
4. **Workflow Sharing**: Workflows may expose patterns/logic (not credentials)

### Future Enhancements

- [ ] Workflow signing and verification
- [ ] MCP server sandboxing
- [ ] Network policy enforcement
- [ ] Encrypted workflow storage
- [ ] Role-based access control (RBAC)
- [ ] Audit log encryption

## Security Testing

GoFlow undergoes regular security testing:

- **Static Analysis**: gosec, golangci-lint security checks
- **Dependency Scanning**: Automated vulnerability scanning
- **Fuzzing**: Critical parsing functions fuzzed
- **Penetration Testing**: Manual security reviews
- **Code Review**: All PRs reviewed for security issues

See `SECURITY_REPORT.md` for latest security audit results.

## Vulnerability Disclosure Policy

We follow **coordinated disclosure**:

1. Reporter notifies us privately
2. We develop and test patch
3. We coordinate release timing with reporter
4. Patch released with security advisory
5. Reporter credited (unless anonymous requested)

## Security Hall of Fame

We recognize security researchers who help improve GoFlow:

<!-- Will be populated as vulnerabilities are reported and fixed -->

*No vulnerabilities reported yet.*

## Additional Resources

- **Architecture**: [CLAUDE.md](CLAUDE.md#security-model)
- **Path Validation**: [CLAUDE.md](CLAUDE.md#path-validation-pkgvalidation)
- **Expression Safety**: [CLAUDE.md](CLAUDE.md#expression-evaluation-security)
- **Security Report**: [SECURITY_REPORT.md](SECURITY_REPORT.md)

## Contact

- **Security Email**: security@[yourdomainhere] (Update this!)
- **GitHub Security**: https://github.com/dshills/goflow/security/advisories
- **General Issues**: https://github.com/dshills/goflow/issues

---

**Thank you for helping keep GoFlow and its users safe!**
