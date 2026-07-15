function columnIndex(columns, field) {
  return columns.findIndex((c) => c.name === field);
}

function rowsAsObjects(data) {
  if (!data) return [];
  return data.rows.map((row) => {
    const obj = {};
    data.columns.forEach((col, i) => {
      obj[col.name] = row[i];
    });
    return obj;
  });
}

function specToEcharts(spec, data) {
  const records = rowsAsObjects(data);
  if (!spec || spec.type === 'table' || !records.length) {
    return { title: { text: 'No chart' } };
  }

  if (spec.type === 'metric') {
    const field = spec.value_field || data.columns[0]?.name;
    const value = records[0]?.[field];
    return {
      title: { text: spec.title || spec.value_label || field },
      series: [{ type: 'gauge', data: [{ value: Number(value) || 0, name: spec.value_label || field }] }],
    };
  }

  const xField = spec.x?.field;
  const yField = spec.y?.field;
  const xValues = records.map((r) => r[xField]);
  const yValues = yField ? records.map((r) => Number(r[yField]) || 0) : [];

  if (spec.type === 'pie') {
    const field = xField || data.columns[0]?.name;
    const counts = {};
    records.forEach((r) => {
      const key = String(r[field]);
      counts[key] = (counts[key] || 0) + 1;
    });
    return {
      title: { text: spec.title || 'Pie' },
      tooltip: { trigger: 'item' },
      series: [{ type: 'pie', radius: '60%', data: Object.entries(counts).map(([name, value]) => ({ name, value })) }],
    };
  }

  const seriesType = spec.type === 'line' ? 'line' : spec.type === 'scatter' ? 'scatter' : 'bar';
  return {
    title: { text: spec.title || seriesType },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: xValues },
    yAxis: { type: 'value' },
    series: [{ type: seriesType, data: yValues }],
  };
}

window.specToEcharts = specToEcharts;
