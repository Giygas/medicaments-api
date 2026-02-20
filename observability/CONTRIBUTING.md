# Contributing to Observability Stack

Thank you for your interest in contributing! This document provides guidelines for contributing to this project.

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Create a feature branch
4. Make your changes
5. Submit a pull request

## Development Workflow

```bash
# Clone your fork
git clone https://github.com/your-username/observability-stack.git
cd observability-stack

# Add upstream remote
git remote add upstream https://github.com/original-owner/observability-stack.git

# Create feature branch
git checkout -b feature/your-feature-name

# Make your changes and test
make setup
make up

# Commit changes
git add .
git commit -m "Add your feature"

# Push to your fork
git push origin feature/your-feature-name
```

## Code Style

### Docker Compose Files

- Use consistent indentation (2 spaces)
- Group related services together
- Add comments for complex configurations
- Use environment variables for configurable values

### Makefile

- Use `.PHONY` for all targets
- Keep targets simple and focused
- Add help text for complex operations
- Use variables for common commands

### Configuration Files

- Use YAML with consistent formatting
- Add comments for non-obvious settings
- Keep default values sensible
- Document all environment variables

## Testing Changes

Before submitting a PR:

```bash
# Test local mode
make setup
make up
# Verify all services start correctly

# Check logs
make logs

# Test Grafana dashboards
# Open http://localhost:3000

# Stop and clean up
make down
docker system prune -f
```

## Types of Contributions

### Bug Fixes

- Clearly describe the bug
- Explain how to reproduce it
- Include the fix with tests (if applicable)
- Update documentation if needed

### New Features

- Explain the use case
- Discuss implementation approach first (if major)
- Follow existing patterns
- Update README and documentation

### Documentation

- Keep documentation clear and concise
- Include examples where helpful
- Update related sections
- Check for broken links

### Dashboards

- Follow existing dashboard structure
- Use meaningful panel titles
- Include proper descriptions
- Test with sample data

## Pull Request Process

1. **Title**: Use clear, descriptive titles
   - `feat: add WireGuard tunnel support`
   - `fix: correct Prometheus retention period`
   - `docs: update setup instructions`

2. **Description**: Include:
   - What changes were made
   - Why they were made
   - How they were tested
   - Any breaking changes

3. **Checklist**:
   - [ ] Code follows project style
   - [ ] Changes are tested
   - [ ] Documentation is updated
   - [ ] No unnecessary files included
   - [ ] Commits are clean and atomic

4. **Review Process**:
   - Automated checks must pass
   - At least one maintainer approval
   - Address review feedback
   - Squash commits if requested

## Project Structure

```
observability-stack/
├── configs/               # Configuration files
│   ├── loki/             # Loki configs
│   ├── prometheus/       # Prometheus configs
│   └── grafana/          # Grafana configs
├── tunnels/             # Tunnel configurations
│   ├── cloudflare.yml
│   ├── tailscale.yml
│   └── wireguard.yml
├── docs/                # Documentation
├── docker-compose.yml   # Main compose file
├── Makefile            # Build commands
├── .env.example        # Environment variables
└── README.md           # Main documentation
```

## Environment Variables

When adding new environment variables:

1. Add to `.env.example` with comments
2. Update README with description
3. Document default values
4. Note which services use the variable

Example:

```bash
# ── New Service ────────────────────────────────────────
NEW_SERVICE_PORT=8080     # Port for new service
NEW_SERVICE_ENABLED=false # Enable/disable feature
```

## Dashboards

When adding new dashboards:

1. Place in `configs/grafana/dashboards/`
2. Use descriptive JSON filename
3. Include `uid` field for uniqueness
4. Add proper tags for filtering
5. Document in README

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

## Breaking Changes

If your change breaks existing functionality:

1. Update documentation clearly
2. Provide migration guide
3. Update version in PR title
4. Allow discussion in PR

## Questions or Issues?

- Open an issue for bugs or feature requests
- Start a discussion for questions
- Join the community chat (if available)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing! 🚀
