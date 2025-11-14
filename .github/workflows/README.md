# GitHub Actions Workflows

Este diretório contém os workflows de CI/CD para o projeto Krakovia Blockchain.

## Workflows Disponíveis

### CI (`ci.yml`)

Pipeline completo de integração contínua que executa em pushes e pull requests para as branches `main` e `develop`.

#### Jobs

**1. Test**
- Executa todos os testes com race detector
- Gera relatório de cobertura de código
- Envia cobertura para Codecov (se configurado)
- Timeout: 30s para os testes

**2. Build**
- Executa somente se os testes passarem
- Gera builds para múltiplas plataformas:
  - Linux (amd64, arm64)
  - macOS/Darwin (amd64, arm64)
  - Windows (amd64)
- Aplica otimizações de tamanho (`-ldflags="-s -w"`)
- Upload dos binários como artefatos (30 dias de retenção)

**3. Lint**
- Executa golangci-lint com todas as verificações
- Garante qualidade e consistência do código
- Timeout: 5 minutos

## Como Usar

Os workflows são executados automaticamente quando você:
- Faz push para `main` ou `develop`
- Cria um Pull Request para `main` ou `develop`

### Download de Artefatos

Após o build ser concluído, você pode baixar os binários:

1. Vá para a aba **Actions** no GitHub
2. Selecione o workflow run desejado
3. Na seção **Artifacts**, baixe o binário da sua plataforma

Os artefatos ficam disponíveis por 30 dias.

## Badges de Status

Adicione ao README.md principal:

```markdown
![CI](https://github.com/SEU_USUARIO/krakovia/actions/workflows/ci.yml/badge.svg)
```

## Requisitos

- Go 1.21 ou superior
- Testes devem completar em menos de 30 segundos
- Código deve passar no golangci-lint

## Configuração Local

Para rodar as mesmas verificações localmente:

```bash
# Testes
go test ./tests/... -v -timeout 30s -race -coverprofile=coverage.out

# Build
go build -o build/krakovia-node -ldflags="-s -w" ./cmd/node

# Lint (instalar golangci-lint primeiro)
golangci-lint run --timeout=5m
```

## Notas

- Os testes usam portas aleatórias (9000-29000) para evitar conflitos
- Diretórios temporários são criados e limpos automaticamente
- Builds são otimizados para tamanho reduzido
- Cache do Go é habilitado para builds mais rápidos
