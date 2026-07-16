const els = {
  question: document.getElementById('question'),
  visualize: document.getElementById('visualize'),
  autoTrain: document.getElementById('auto_train'),
  askBtn: document.getElementById('askBtn'),
  sessionMeta: document.getElementById('sessionMeta'),
  error: document.getElementById('error'),
  sql: document.getElementById('sql'),
  table: document.getElementById('table'),
  chart: document.getElementById('chart'),
  trainDDL: document.getElementById('trainDDL'),
  trainDoc: document.getElementById('trainDoc'),
  trainBtn: document.getElementById('trainBtn'),
  trainingList: document.getElementById('trainingList'),
};

let sessionId = '';
let chartInstance = echarts.init(els.chart);

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...(sessionId ? { 'X-Session-ID': sessionId } : {}) },
    ...options,
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

function renderTable(data) {
  if (!data || !data.columns || !data.rows) {
    els.table.textContent = '无数据';
    return;
  }
  const headers = data.columns.map((c) => c.name);
  const rows = data.rows.map((row) => `<tr>${row.map((c) => `<td>${c ?? ''}</td>`).join('')}</tr>`).join('');
  els.table.innerHTML = `<table><thead><tr>${headers.map((h) => `<th>${h}</th>`).join('')}</tr></thead><tbody>${rows}</tbody></table>`;
}

function renderChart(spec, data) {
  if (!spec || spec.type === 'table' || !window.specToEcharts) {
    chartInstance.clear();
    return;
  }
  const option = window.specToEcharts(spec, data);
  chartInstance.setOption(option, true);
}

els.askBtn.addEventListener('click', async () => {
  els.error.textContent = '';
  try {
    const result = await api('/api/v1/ask', {
      method: 'POST',
      body: JSON.stringify({
        question: els.question.value,
        visualize: els.visualize.checked,
        auto_train: els.autoTrain.checked,
      }),
    });
    sessionId = result.session_id;
    els.sessionMeta.textContent = `session: ${sessionId}`;
    els.sql.textContent = result.sql || '';
    renderTable(result.data);
    renderChart(result.chart, result.data);
  } catch (err) {
    els.error.textContent = String(err.message || err);
  }
});

els.trainBtn.addEventListener('click', async () => {
  els.error.textContent = '';
  try {
    await api('/api/v1/train', {
      method: 'POST',
      body: JSON.stringify({
        ddl: els.trainDDL.value || undefined,
        documentation: els.trainDoc.value || undefined,
      }),
    });
    await loadTraining();
  } catch (err) {
    els.error.textContent = String(err.message || err);
  }
});

async function loadTraining() {
  const data = await api('/api/v1/training_data');
  els.trainingList.textContent = JSON.stringify(data.items, null, 2);
}

loadTraining().catch(() => {});
