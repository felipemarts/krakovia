package api

// htmlUI contém a interface HTML/JavaScript para interagir com a API
const htmlUI = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Krakovia Node Dashboard</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        h1 {
            color: white;
            text-align: center;
            margin-bottom: 30px;
            font-size: 2.5em;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }

        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }

        .card {
            background: white;
            border-radius: 10px;
            padding: 20px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }

        .card h2 {
            color: #667eea;
            margin-bottom: 15px;
            font-size: 1.5em;
            border-bottom: 2px solid #667eea;
            padding-bottom: 10px;
        }

        .stat {
            display: flex;
            justify-content: space-between;
            padding: 10px 0;
            border-bottom: 1px solid #eee;
        }

        .stat:last-child {
            border-bottom: none;
        }

        .stat-label {
            font-weight: 600;
            color: #555;
        }

        .stat-value {
            color: #333;
            font-family: 'Courier New', monospace;
        }

        .form-group {
            margin-bottom: 15px;
        }

        label {
            display: block;
            margin-bottom: 5px;
            font-weight: 600;
            color: #555;
        }

        input, textarea {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 14px;
        }

        button {
            background: #667eea;
            color: white;
            border: none;
            padding: 12px 24px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            width: 100%;
            transition: background 0.3s;
        }

        button:hover {
            background: #5568d3;
        }

        button:disabled {
            background: #ccc;
            cursor: not-allowed;
        }

        .btn-danger {
            background: #dc3545;
        }

        .btn-danger:hover {
            background: #c82333;
        }

        .btn-success {
            background: #28a745;
        }

        .btn-success:hover {
            background: #218838;
        }

        .message {
            padding: 12px;
            border-radius: 5px;
            margin-top: 15px;
            display: none;
        }

        .message.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
            display: block;
        }

        .message.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
            display: block;
        }

        .block-list {
            max-height: 300px;
            overflow-y: auto;
        }

        .block-item {
            padding: 10px;
            background: #f8f9fa;
            margin-bottom: 10px;
            border-radius: 5px;
            border-left: 3px solid #667eea;
        }

        .peers-list {
            max-height: 200px;
            overflow-y: auto;
        }

        .peer-item {
            padding: 8px;
            background: #f8f9fa;
            margin-bottom: 5px;
            border-radius: 5px;
            font-family: 'Courier New', monospace;
            font-size: 12px;
        }

        .badge {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 3px;
            font-size: 11px;
            font-weight: 600;
        }

        .badge-success {
            background: #28a745;
            color: white;
        }

        .badge-danger {
            background: #dc3545;
            color: white;
        }

        .refresh-btn {
            background: #6c757d;
            padding: 8px 16px;
            font-size: 14px;
            margin-bottom: 15px;
        }

        .refresh-btn:hover {
            background: #5a6268;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Krakovia Node Dashboard</h1>

        <div class="grid">
            <!-- Status do Nó -->
            <div class="card">
                <h2>Status do Nó</h2>
                <button class="refresh-btn" onclick="loadStatus()">Atualizar</button>
                <div id="node-status">
                    <div class="stat">
                        <span class="stat-label">ID:</span>
                        <span class="stat-value" id="node-id">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Altura:</span>
                        <span class="stat-value" id="chain-height">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Mempool:</span>
                        <span class="stat-value" id="mempool-size">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Peers:</span>
                        <span class="stat-value" id="peers-count">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Minerando:</span>
                        <span class="stat-value" id="mining-status">-</span>
                    </div>
                </div>
            </div>

            <!-- Carteira -->
            <div class="card">
                <h2>Carteira</h2>
                <button class="refresh-btn" onclick="loadWallet()">Atualizar</button>
                <div id="wallet-info">
                    <div class="stat">
                        <span class="stat-label">Endereço:</span>
                        <span class="stat-value" id="wallet-address" style="font-size: 10px; word-break: break-all; cursor: pointer;" onclick="copyAddress()" title="Clique para copiar">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Saldo:</span>
                        <span class="stat-value" id="wallet-balance">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Stake:</span>
                        <span class="stat-value" id="wallet-stake">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Nonce:</span>
                        <span class="stat-value" id="wallet-nonce">-</span>
                    </div>
                </div>
            </div>

            <!-- Último Bloco -->
            <div class="card">
                <h2>Último Bloco</h2>
                <button class="refresh-btn" onclick="loadLastBlock()">Atualizar</button>
                <div id="last-block">
                    <div class="stat">
                        <span class="stat-label">Altura:</span>
                        <span class="stat-value" id="block-height">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Hash:</span>
                        <span class="stat-value" id="block-hash" style="font-size: 10px; word-break: break-all;">-</span>
                    </div>
                    <div class="stat">
                        <span class="stat-label">Transações:</span>
                        <span class="stat-value" id="block-txs">-</span>
                    </div>
                </div>
            </div>
        </div>

        <div class="grid">
            <!-- Transferência -->
            <div class="card">
                <h2>Transferir</h2>
                <form id="transfer-form" onsubmit="return handleTransfer(event)">
                    <div class="form-group">
                        <label for="transfer-to">Destinatário:</label>
                        <input type="text" id="transfer-to" required placeholder="Endereço hex...">
                    </div>
                    <div class="form-group">
                        <label for="transfer-amount">Quantidade:</label>
                        <input type="number" id="transfer-amount" required min="1" placeholder="100">
                    </div>
                    <div class="form-group">
                        <label for="transfer-fee">Taxa:</label>
                        <input type="number" id="transfer-fee" value="10" min="0" placeholder="10">
                    </div>
                    <div class="form-group">
                        <label for="transfer-data">Dados (opcional):</label>
                        <input type="text" id="transfer-data" placeholder="Mensagem...">
                    </div>
                    <button type="submit">Enviar Transferência</button>
                    <div id="transfer-message" class="message"></div>
                </form>
            </div>

            <!-- Stake -->
            <div class="card">
                <h2>Stake</h2>
                <form id="stake-form" onsubmit="return handleStake(event)">
                    <div class="form-group">
                        <label for="stake-amount">Quantidade:</label>
                        <input type="number" id="stake-amount" required min="1" placeholder="1000">
                    </div>
                    <div class="form-group">
                        <label for="stake-fee">Taxa:</label>
                        <input type="number" id="stake-fee" value="10" min="0" placeholder="10">
                    </div>
                    <button type="submit" class="btn-success">Fazer Stake</button>
                    <div id="stake-message" class="message"></div>
                </form>

                <hr style="margin: 20px 0; border: none; border-top: 1px solid #eee;">

                <form id="unstake-form" onsubmit="return handleUnstake(event)">
                    <div class="form-group">
                        <label for="unstake-amount">Quantidade:</label>
                        <input type="number" id="unstake-amount" required min="1" placeholder="1000">
                    </div>
                    <div class="form-group">
                        <label for="unstake-fee">Taxa:</label>
                        <input type="number" id="unstake-fee" value="10" min="0" placeholder="10">
                    </div>
                    <button type="submit" class="btn-danger">Fazer Unstake</button>
                    <div id="unstake-message" class="message"></div>
                </form>
            </div>

            <!-- Mineração -->
            <div class="card">
                <h2>Mineração</h2>
                <p style="margin-bottom: 15px; color: #666;">Controle a mineração de blocos neste nó.</p>
                <button onclick="startMining()" class="btn-success" style="margin-bottom: 10px;">Iniciar Mineração</button>
                <button onclick="stopMining()" class="btn-danger">Parar Mineração</button>
                <div id="mining-message" class="message"></div>
            </div>
        </div>

        <div class="grid">
            <!-- Peers -->
            <div class="card">
                <h2>Peers Conectados</h2>
                <button class="refresh-btn" onclick="loadPeers()">Atualizar</button>
                <div id="peers-list" class="peers-list">
                    <p style="color: #999;">Carregando...</p>
                </div>
            </div>
        </div>
    </div>

    <script>
        // Configuração da API
        const API_BASE = '';
        let authToken = null;

        // Função para fazer requisições autenticadas
        async function apiRequest(endpoint, options = {}) {
            if (!options.headers) {
                options.headers = {};
            }

            // Adicionar autenticação básica se disponível
            if (authToken) {
                options.headers['Authorization'] = 'Basic ' + authToken;
            }

            const response = await fetch(API_BASE + endpoint, options);

            if (response.status === 401) {
                // Solicitar credenciais
                requestAuth();
                throw new Error('Autenticação necessária');
            }

            return response;
        }

        // Solicitar autenticação
        function requestAuth() {
            const username = prompt('Usuário:');
            const password = prompt('Senha:');

            if (username && password) {
                authToken = btoa(username + ':' + password);
                loadAll();
            }
        }

        // Carregar status do nó
        async function loadStatus() {
            try {
                const response = await fetch(API_BASE + '/api/status');
                const data = await response.json();

                document.getElementById('node-id').textContent = data.node_id || '-';
                document.getElementById('chain-height').textContent = data.chain_height || '0';
                document.getElementById('mempool-size').textContent = data.mempool_size || '0';
                document.getElementById('peers-count').textContent = data.peers_count || '0';
                document.getElementById('mining-status').textContent = data.mining ? 'Sim' : 'Não';
            } catch (error) {
                console.error('Erro ao carregar status:', error);
            }
        }

        // Carregar informações da carteira
        async function loadWallet() {
            try {
                const response = await apiRequest('/api/wallet');
                const data = await response.json();

                document.getElementById('wallet-address').textContent = data.address || '-';
                document.getElementById('wallet-balance').textContent = data.balance || '0';
                document.getElementById('wallet-stake').textContent = data.stake || '0';
                document.getElementById('wallet-nonce').textContent = data.nonce || '0';
            } catch (error) {
                console.error('Erro ao carregar carteira:', error);
            }
        }

        // Copiar endereço da carteira
        function copyAddress() {
            const address = document.getElementById('wallet-address').textContent;
            if (address && address !== '-') {
                navigator.clipboard.writeText(address).then(() => {
                    alert('Endereço copiado para a área de transferência!');
                }).catch(err => {
                    console.error('Erro ao copiar:', err);
                });
            }
        }

        // Carregar último bloco
        async function loadLastBlock() {
            try {
                const response = await fetch(API_BASE + '/api/lastblock');
                const data = await response.json();

                document.getElementById('block-height').textContent = data.height || '0';
                document.getElementById('block-hash').textContent = data.hash || '-';
                document.getElementById('block-txs').textContent = data.tx_count || '0';
            } catch (error) {
                console.error('Erro ao carregar último bloco:', error);
            }
        }

        // Carregar peers
        async function loadPeers() {
            try {
                const response = await fetch(API_BASE + '/api/peers');
                const data = await response.json();

                const peersList = document.getElementById('peers-list');

                if (data.count === 0) {
                    peersList.innerHTML = '<p style="color: #999;">Nenhum peer conectado</p>';
                    return;
                }

                peersList.innerHTML = data.peers.map(peer =>
                    '<div class="peer-item">' +
                    peer.id +
                    ' <span class="badge badge-' + (peer.ready ? 'success' : 'danger') + '">' +
                    (peer.ready ? 'Pronto' : 'Conectando') +
                    '</span></div>'
                ).join('');
            } catch (error) {
                console.error('Erro ao carregar peers:', error);
            }
        }

        // Handler de transferência
        async function handleTransfer(event) {
            event.preventDefault();

            const messageEl = document.getElementById('transfer-message');
            messageEl.className = 'message';
            messageEl.style.display = 'none';

            const data = {
                to: document.getElementById('transfer-to').value,
                amount: parseInt(document.getElementById('transfer-amount').value),
                fee: parseInt(document.getElementById('transfer-fee').value),
                data: document.getElementById('transfer-data').value
            };

            try {
                const response = await apiRequest('/api/transaction/send', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (response.ok) {
                    messageEl.textContent = 'Transação enviada! ID: ' + result.tx_id.substring(0, 16) + '...';
                    messageEl.className = 'message success';
                    messageEl.style.display = 'block';
                    document.getElementById('transfer-form').reset();
                    setTimeout(() => { loadWallet(); loadStatus(); }, 1000);
                } else {
                    messageEl.textContent = result.error || 'Erro ao enviar transação';
                    messageEl.className = 'message error';
                    messageEl.style.display = 'block';
                }
            } catch (error) {
                messageEl.textContent = 'Erro: ' + error.message;
                messageEl.className = 'message error';
                messageEl.style.display = 'block';
            }

            return false;
        }

        // Handler de stake
        async function handleStake(event) {
            event.preventDefault();

            const messageEl = document.getElementById('stake-message');
            messageEl.className = 'message';
            messageEl.style.display = 'none';

            const data = {
                amount: parseInt(document.getElementById('stake-amount').value),
                fee: parseInt(document.getElementById('stake-fee').value)
            };

            try {
                const response = await apiRequest('/api/transaction/stake', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (response.ok) {
                    messageEl.textContent = 'Stake realizado! ID: ' + result.tx_id.substring(0, 16) + '...';
                    messageEl.className = 'message success';
                    messageEl.style.display = 'block';
                    document.getElementById('stake-form').reset();
                    setTimeout(() => { loadWallet(); loadStatus(); }, 1000);
                } else {
                    messageEl.textContent = result.error || 'Erro ao fazer stake';
                    messageEl.className = 'message error';
                    messageEl.style.display = 'block';
                }
            } catch (error) {
                messageEl.textContent = 'Erro: ' + error.message;
                messageEl.className = 'message error';
                messageEl.style.display = 'block';
            }

            return false;
        }

        // Handler de unstake
        async function handleUnstake(event) {
            event.preventDefault();

            const messageEl = document.getElementById('unstake-message');
            messageEl.className = 'message';
            messageEl.style.display = 'none';

            const data = {
                amount: parseInt(document.getElementById('unstake-amount').value),
                fee: parseInt(document.getElementById('unstake-fee').value)
            };

            try {
                const response = await apiRequest('/api/transaction/unstake', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (response.ok) {
                    messageEl.textContent = 'Unstake realizado! ID: ' + result.tx_id.substring(0, 16) + '...';
                    messageEl.className = 'message success';
                    messageEl.style.display = 'block';
                    document.getElementById('unstake-form').reset();
                    setTimeout(() => { loadWallet(); loadStatus(); }, 1000);
                } else {
                    messageEl.textContent = result.error || 'Erro ao fazer unstake';
                    messageEl.className = 'message error';
                    messageEl.style.display = 'block';
                }
            } catch (error) {
                messageEl.textContent = 'Erro: ' + error.message;
                messageEl.className = 'message error';
                messageEl.style.display = 'block';
            }

            return false;
        }

        // Iniciar mineração
        async function startMining() {
            const messageEl = document.getElementById('mining-message');
            messageEl.className = 'message';
            messageEl.style.display = 'none';

            try {
                const response = await apiRequest('/api/mining/start', { method: 'POST' });
                const result = await response.json();

                if (response.ok) {
                    messageEl.textContent = 'Mineração iniciada!';
                    messageEl.className = 'message success';
                    setTimeout(loadStatus, 500);
                } else {
                    messageEl.textContent = result.error || 'Erro ao iniciar mineração';
                    messageEl.className = 'message error';
                }
            } catch (error) {
                messageEl.textContent = 'Erro: ' + error.message;
                messageEl.className = 'message error';
            }
        }

        // Parar mineração
        async function stopMining() {
            const messageEl = document.getElementById('mining-message');
            messageEl.className = 'message';
            messageEl.style.display = 'none';

            try {
                const response = await apiRequest('/api/mining/stop', { method: 'POST' });
                const result = await response.json();

                if (response.ok) {
                    messageEl.textContent = 'Mineração parada!';
                    messageEl.className = 'message success';
                    setTimeout(loadStatus, 500);
                } else {
                    messageEl.textContent = result.error || 'Erro ao parar mineração';
                    messageEl.className = 'message error';
                }
            } catch (error) {
                messageEl.textContent = 'Erro: ' + error.message;
                messageEl.className = 'message error';
            }
        }

        // Carregar tudo
        function loadAll() {
            loadStatus();
            loadWallet();
            loadLastBlock();
            loadPeers();
        }

        // Auto-refresh a cada 5 segundos
        setInterval(loadStatus, 5000);
        setInterval(loadLastBlock, 5000);

        // Carregar ao iniciar
        loadAll();
    </script>
</body>
</html>
`
