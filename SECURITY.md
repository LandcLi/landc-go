# Security Policy

## Supported Versions

We currently support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of **landc-go** seriously. If you believe you have found a
security vulnerability, please **do not** open a public issue.

Instead, report it privately via one of the following methods:

- **GitHub Security Advisory**: Navigate to the repository's
  [Security Advisories](https://github.com/LandcLi/landc-go/security/advisories)
  tab and click "Report a vulnerability"
- **Email**: security@landcli.dev

### What to include in your report

- Type of vulnerability (e.g., SQL injection, XSS, privilege escalation)
- Full paths of source files related to the issue
- Steps to reproduce (preferably a minimal proof-of-concept)
- Potential impact
- Any suggested fix (if available)

### What to expect

1. **Acknowledgment**: We will acknowledge receipt within 48 hours
2. **Investigation**: We will investigate and provide an initial assessment
   within 5 business days
3. **Fix timeline**: We will work on a fix and release timeline based on severity:
   - **Critical**: Within 7 days
   - **High**: Within 14 days
   - **Medium/Low**: Within 30 days
4. **Disclosure**: Once a fix is released, we will coordinate public disclosure

We appreciate your help in keeping **landc-go** and its users safe.

## Security Best Practices for Users

When using landc-go in production, follow these security practices:

1. **Keep dependencies updated**: Regularly run `go mod tidy` and update Go modules
2. **Use trusted proxies**: Configure `TRUSTED_PROXIES` environment variable with
   appropriate CIDR ranges to prevent IP spoofing
3. **Enable security middleware**: Use the built-in CORS, CSRF, and security
   headers middleware
4. **Rotate secrets**: Regularly rotate JWT secrets, session keys, and database
   passwords
5. **Use HTTPS in production**: Never expose services over plain HTTP in production
6. **Limit rate**: Enable rate limiting on public-facing endpoints
7. **Audit logs**: Enable request logging and monitor for suspicious patterns
