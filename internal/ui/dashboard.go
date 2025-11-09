package ui

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PMP Mock HTTP - Request Dashboard</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://code.jquery.com/jquery-3.7.1.min.js"></script>
</head>
<body class="bg-gray-100">
    <div class="container mx-auto px-4 py-8">
        <div class="bg-white rounded-lg shadow-md p-6 mb-6">
            <div class="flex justify-between items-center">
                <div>
                    <h1 class="text-3xl font-bold text-gray-800">PMP Mock HTTP</h1>
                    <p class="text-gray-600 mt-1">Request Dashboard</p>
                </div>
                <div class="flex gap-4">
                    <div class="text-right">
                        <div class="text-2xl font-bold text-blue-600" id="total-requests">0</div>
                        <div class="text-sm text-gray-600">Total Requests</div>
                    </div>
                    <div class="text-right">
                        <div class="text-2xl font-bold text-green-600" id="matched-requests">0</div>
                        <div class="text-sm text-gray-600">Matched</div>
                    </div>
                    <div class="text-right">
                        <div class="text-2xl font-bold text-red-600" id="unmatched-requests">0</div>
                        <div class="text-sm text-gray-600">Unmatched</div>
                    </div>
                </div>
            </div>
            <div class="mt-4 flex gap-2">
                <button id="refresh-btn" class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">Refresh Now</button>
                <button id="clear-btn" class="bg-red-500 hover:bg-red-700 text-white font-bold py-2 px-4 rounded">Clear All</button>
                <label class="flex items-center ml-4">
                    <input type="checkbox" id="auto-refresh" checked class="mr-2">
                    <span class="text-gray-700">Auto-refresh (2s)</span>
                </label>
            </div>
        </div>
        <div class="bg-white rounded-lg shadow-md p-6">
            <h2 class="text-xl font-bold text-gray-800 mb-4">Recent Requests</h2>
            <div id="requests-container"><p class="text-gray-500 text-center py-8">Loading requests...</p></div>
        </div>
    </div>
    <script>
        let autoRefreshInterval = null;
        function fetchRequests() {
            $.get('/api/requests', function(data) {
                renderRequests(data);
                updateStats(data);
            }).fail(function() {
                $('#requests-container').html('<p class="text-red-500 text-center py-8">Failed to load requests</p>');
            });
        }
        function updateStats(requests) {
            const total = requests.length;
            const matched = requests.filter(r => r.matched).length;
            const unmatched = total - matched;
            $('#total-requests').text(total);
            $('#matched-requests').text(matched);
            $('#unmatched-requests').text(unmatched);
        }
        function renderRequests(requests) {
            if (requests.length === 0) {
                $('#requests-container').html('<p class="text-gray-500 text-center py-8">No requests yet</p>');
                return;
            }
            let html = '';
            requests.forEach(function(req) {
                const matchedClass = req.matched ? 'border-green-500' : 'border-red-500';
                const matchedBadge = req.matched
                    ? '<span class="bg-green-100 text-green-800 text-xs font-semibold px-2.5 py-0.5 rounded">MATCHED</span>'
                    : '<span class="bg-red-100 text-red-800 text-xs font-semibold px-2.5 py-0.5 rounded">UNMATCHED</span>';
                const timestamp = new Date(req.timestamp).toLocaleString();
                const statusClass = req.status_code >= 200 && req.status_code < 300 ? 'text-green-600' :
                                   req.status_code >= 400 ? 'text-red-600' : 'text-yellow-600';
                html += '<div class="border-l-4 ' + matchedClass + ' bg-gray-50 p-4 mb-4 rounded">';
                html += '  <div class="flex justify-between items-start mb-2">';
                html += '    <div class="flex items-center gap-2">';
                html += '      <span class="font-bold text-lg">' + escapeHtml(req.method) + '</span>';
                html += '      <span class="text-gray-700">' + escapeHtml(req.uri) + '</span>';
                html += '      ' + matchedBadge;
                html += '    </div>';
                html += '    <div class="text-right">';
                html += '      <div class="text-sm text-gray-600">' + timestamp + '</div>';
                html += '      <div class="text-sm text-gray-500">' + escapeHtml(req.remote_addr) + '</div>';
                html += '    </div>';
                html += '  </div>';
                if (req.matched && req.mock_name) {
                    html += '  <div class="mb-2"><span class="text-sm text-gray-600">Mock: </span>';
                    html += '    <span class="text-sm font-semibold text-blue-600">' + escapeHtml(req.mock_name) + '</span></div>';
                }
                html += '  <div class="mb-2"><span class="text-sm text-gray-600">Status: </span>';
                html += '    <span class="text-sm font-semibold ' + statusClass + '">' + req.status_code + '</span></div>';
                if (req.headers && Object.keys(req.headers).length > 0) {
                    html += '  <details class="mt-2"><summary class="text-sm font-semibold text-gray-700 cursor-pointer">Headers</summary>';
                    html += '    <div class="bg-white p-2 mt-1 rounded text-xs font-mono">';
                    Object.keys(req.headers).forEach(function(key) {
                        html += '      <div><span class="text-gray-600">' + escapeHtml(key) + ':</span> ' + escapeHtml(req.headers[key]) + '</div>';
                    });
                    html += '    </div></details>';
                }
                if (req.body) {
                    html += '  <details class="mt-2"><summary class="text-sm font-semibold text-gray-700 cursor-pointer">Request Body</summary>';
                    html += '    <pre class="bg-white p-2 mt-1 rounded text-xs overflow-x-auto">' + escapeHtml(req.body) + '</pre></details>';
                }
                if (req.response) {
                    html += '  <details class="mt-2"><summary class="text-sm font-semibold text-gray-700 cursor-pointer">Response</summary>';
                    html += '    <pre class="bg-white p-2 mt-1 rounded text-xs overflow-x-auto">' + escapeHtml(req.response) + '</pre></details>';
                }
                html += '</div>';
            });
            $('#requests-container').html(html);
        }
        function escapeHtml(text) {
            if (!text) return '';
            const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
            return text.toString().replace(/[&<>"']/g, function(m) { return map[m]; });
        }
        function clearRequests() {
            if (confirm('Are you sure you want to clear all request logs?')) {
                $.post('/api/clear', function() { fetchRequests(); }).fail(function() { alert('Failed to clear requests'); });
            }
        }
        function updateAutoRefresh() {
            if ($('#auto-refresh').is(':checked')) {
                if (!autoRefreshInterval) {
                    autoRefreshInterval = setInterval(fetchRequests, 2000);
                }
            } else {
                if (autoRefreshInterval) {
                    clearInterval(autoRefreshInterval);
                    autoRefreshInterval = null;
                }
            }
        }
        $(document).ready(function() {
            $('#refresh-btn').click(fetchRequests);
            $('#clear-btn').click(clearRequests);
            $('#auto-refresh').change(updateAutoRefresh);
            fetchRequests();
            updateAutoRefresh();
        });
    </script>
</body>
</html>
`
