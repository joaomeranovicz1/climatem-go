document.addEventListener('DOMContentLoaded', () => {
    iniciarApp();
    
    // enter no input 
    const inputCidade = document.getElementById('input-cidade');
    if (inputCidade) {
        inputCidade.addEventListener('keypress', function (e) {
            if (e.key === 'Enter') {
                buscarCidadeBotao();
            }
        });
    }
});

function iniciarApp() {
    const statusEl = document.getElementById('status-conexao');
    if (navigator.geolocation) {
        statusEl.textContent = '📍 GPS...';
        navigator.geolocation.getCurrentPosition(sucessoGPS, erroGPS);
    } else {
        erroGPS("Sem suporte");
    }
}

function sucessoGPS(position) {
    const titulo = document.getElementById('nome-cidade-titulo');
    if (titulo) titulo.textContent = "SUA LOCALIZAÇÃO";
    
    buscarDados(position.coords.latitude, position.coords.longitude);
}

function erroGPS(err) {
    console.warn(err);
    const statusEl = document.getElementById('status-conexao');
    if (statusEl) statusEl.textContent = '⚠️ GPS Off';
    
    // Fallback: São Paulo
    buscarDados(-23.55, -46.63); 
}

// busca (nao retornava)- é para estar certo agra
async function buscarCidadeBotao() {
    const input = document.getElementById('input-cidade');
    const nomeCidade = input.value;
    const statusEl = document.getElementById('status-conexao');

    if (!nomeCidade) {
        alert("Digite o nome de uma cidade!");
        return;
    }

    statusEl.textContent = '🔎 Buscando...';

    try {
        // Chama o Go
        const response = await fetch(`/api/cidade?nome=${encodeURIComponent(nomeCidade)}`);
        
        if (!response.ok) {
            alert("Cidade não encontrada!");
            statusEl.textContent = '❌ Erro';
            return;
        }

        const data = await response.json();

        
        let nomeFormatado = data.name;
        if(data.admin1) nomeFormatado += ` - ${data.admin1}`;
        if(data.country) nomeFormatado += ` (${data.country})`;
        
        const titulo = document.getElementById('nome-cidade-titulo');
        if(titulo) titulo.textContent = nomeFormatado;

        // clear input
        input.value = "";

        //  funçao principal
        buscarDados(data.latitude, data.longitude);

    } catch (error) {
        console.error("ERRO:", error);
        alert("Erro ao buscar cidade.");
        statusEl.textContent = '❌ Erro';
    }
}

//  dados
async function buscarDados(lat, lon) {
    const statusEl = document.getElementById('status-conexao');
    if(statusEl) statusEl.textContent = '☁️ Carregando...';

    try {
        const response = await fetch(`/api/clima?lat=${lat}&lon=${lon}`);
        const data = await response.json();

        // Atualiza textos
        document.getElementById('temperatura').textContent = Math.round(data.temp) + '°';
        document.getElementById('descricao').textContent = data.descricao;
        document.getElementById('icone-clima').textContent = data.icone;
        document.getElementById('sensacao').textContent = Math.round(data.sensacao) + '°';
        document.getElementById('umidade').textContent = data.umidade + '%';
        document.getElementById('uv').textContent = data.uv ? data.uv.toFixed(1) : '--';
        
        // Pólen
        const polenEl = document.getElementById('polen');
        if (polenEl) {
            polenEl.textContent = data.polen;
            polenEl.style.color = data.polen.includes("Indisponível") ? "#9ca3af" : "#5a67d8";
        }

        // Dica
        const dicaEl = document.getElementById('box-dica');
        if (dicaEl) {
            dicaEl.textContent = data.dica;
            dicaEl.className = 'tip-box'; // Limpa classes antigas
            
            if (data.tipo_dica === 'perigo') dicaEl.classList.add('dica-perigo');
            else if (data.tipo_dica === 'atencao') dicaEl.classList.add('dica-atencao');
            else dicaEl.classList.add('dica-bom');
        }

        if(statusEl) {
            statusEl.textContent = '🟢 Online';
            statusEl.style.color = '#fff';
        }

    } catch (error) {
        console.error(error);
        if(statusEl) statusEl.textContent = '❌ Erro';
    }
}