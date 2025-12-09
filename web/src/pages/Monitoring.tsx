import { useState, useEffect, useCallback } from 'react';
import {
  Cpu,
  HardDrive,
  MemoryStick,
  Network,
  Server,
  RefreshCw,
  Container,
  Activity,
  AlertTriangle,
  Clock,
  TrendingUp,
  ChevronDown,
  ChevronUp,
  BarChart3,
  Wifi,
  WifiOff,
} from 'lucide-react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Area,
  AreaChart,
  Legend,
} from 'recharts';
import { api } from '@/api/client';
import { API_ENDPOINTS } from '@/lib/config';
import Card from '@/components/ui/Card';
import { useMetricsSSE } from '@/hooks/useMetricsSSE';

interface HistoricalMetric {
  id: number;
  timestamp: string;
  cpu_percent: number;
  memory_percent: number;
  memory_used: number;
  memory_total: number;
  disk_data: string;
  network_data: string;
  load_avg: string;
  uptime: number;
}

interface HistoricalResponse {
  from: string;
  to: string;
  resolution: string;
  data: HistoricalMetric[];
}

interface ContainerHistoricalMetric {
  id: number;
  timestamp: string;
  container_id: string;
  container_name: string;
  image: string;
  state: string;
  cpu_percent: number;
  memory_percent: number;
  memory_used: number;
  memory_limit: number;
  network_rx: number;
  network_tx: number;
  block_read: number;
  block_write: number;
}

interface ContainerHistoricalResponse {
  container_id: string;
  from: string;
  to: string;
  data: ContainerHistoricalMetric[];
}

// Time range options
const TIME_RANGES = [
  { label: '1H', value: '1h', hours: 1 },
  { label: '6H', value: '6h', hours: 6 },
  { label: '24H', value: '24h', hours: 24 },
  { label: '7D', value: '7d', hours: 168 },
  { label: '30D', value: '30d', hours: 720 },
];

// Format bytes to human readable
function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

// Format uptime
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  const parts = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);
  return parts.join(' ') || '< 1m';
}

// Format timestamp for chart
function formatChartTime(timestamp: string, range: string): string {
  const date = new Date(timestamp);
  if (range === '1h' || range === '6h') {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } else if (range === '24h') {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } else {
    return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
  }
}

// Progress bar component
function ProgressBar({
  value,
  max = 100,
  color = 'blue',
  showLabel = true,
}: {
  value: number;
  max?: number;
  color?: string;
  showLabel?: boolean;
}) {
  const percentage = Math.min((value / max) * 100, 100);
  const colorClasses: Record<string, string> = {
    blue: 'bg-blue-500',
    green: 'bg-green-500',
    yellow: 'bg-yellow-500',
    red: 'bg-red-500',
    purple: 'bg-purple-500',
  };

  // Auto color based on value
  let autoColor = 'green';
  if (percentage >= 90) autoColor = 'red';
  else if (percentage >= 70) autoColor = 'yellow';

  const bgColor = colorClasses[color === 'auto' ? autoColor : color] || colorClasses.blue;

  return (
    <div className="w-full">
      <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
        <div
          className={`h-full ${bgColor} transition-all duration-300`}
          style={{ width: `${percentage}%` }}
        />
      </div>
      {showLabel && (
        <div className="text-xs text-gray-500 mt-1 text-right">{percentage.toFixed(1)}%</div>
      )}
    </div>
  );
}

// Time Range Selector Component
function TimeRangeSelector({
  selected,
  onChange,
}: {
  selected: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
      {TIME_RANGES.map((range) => (
        <button
          key={range.value}
          onClick={() => onChange(range.value)}
          className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
            selected === range.value
              ? 'bg-white text-blue-600 shadow-sm'
              : 'text-gray-600 hover:text-gray-900'
          }`}
        >
          {range.label}
        </button>
      ))}
    </div>
  );
}

// Custom tooltip for charts
function CustomTooltip({ active, payload, label }: any) {
  if (active && payload && payload.length) {
    return (
      <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-3">
        <p className="text-xs text-gray-500 mb-2">{label}</p>
        {payload.map((entry: any, index: number) => (
          <p key={index} className="text-sm" style={{ color: entry.color }}>
            {entry.name}: {entry.value.toFixed(1)}%
          </p>
        ))}
      </div>
    );
  }
  return null;
}

// Metric Chart Component
function MetricChart({
  title,
  icon: Icon,
  iconBgColor,
  iconColor,
  data,
  dataKey,
  color,
  timeRange,
  onTimeRangeChange,
  loading,
  yAxisLabel,
  gradientId,
}: {
  title: string;
  icon: any;
  iconBgColor: string;
  iconColor: string;
  data: any[];
  dataKey: string;
  color: string;
  timeRange: string;
  onTimeRangeChange: (value: string) => void;
  loading: boolean;
  yAxisLabel?: string;
  gradientId: string;
}) {
  return (
    <Card className="p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`${iconBgColor} p-2 rounded-lg`}>
            <Icon className={`h-5 w-5 ${iconColor}`} />
          </div>
          <h3 className="text-lg font-medium text-gray-900">{title}</h3>
        </div>
        <TimeRangeSelector selected={timeRange} onChange={onTimeRangeChange} />
      </div>
      <div className="h-64">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : data.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-500">
            <div className="text-center">
              <Clock className="h-8 w-8 mx-auto mb-2 text-gray-400" />
              <p>No historical data available</p>
              <p className="text-sm text-gray-400">Data will appear once collected</p>
            </div>
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                  <stop offset="95%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis
                dataKey="time"
                tick={{ fontSize: 11, fill: '#6b7280' }}
                tickLine={false}
                axisLine={{ stroke: '#e5e7eb' }}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 11, fill: '#6b7280' }}
                tickLine={false}
                axisLine={{ stroke: '#e5e7eb' }}
                tickFormatter={(value) => `${value}%`}
                width={45}
              />
              <Tooltip content={<CustomTooltip />} />
              <Area
                type="monotone"
                dataKey={dataKey}
                stroke={color}
                strokeWidth={2}
                fill={`url(#${gradientId})`}
                name={yAxisLabel || title}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>
    </Card>
  );
}

// Container History Chart Component
function ContainerHistoryChart({
  containerId,
  containerName,
  timeRange,
}: {
  containerId: string;
  containerName: string;
  timeRange: string;
}) {
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [localTimeRange, setLocalTimeRange] = useState(timeRange);

  const fetchHistory = useCallback(async () => {
    setLoading(true);
    try {
      const range = TIME_RANGES.find((r) => r.value === localTimeRange);
      const hours = range?.hours || 1;
      const from = new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
      const to = new Date().toISOString();

      const response = await api.get<ContainerHistoricalResponse>(
        `${API_ENDPOINTS.metricsDockerContainerHistory(containerId)}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
      );

      const chartData = (response.data || []).map((item) => ({
        time: formatChartTime(item.timestamp, localTimeRange),
        timestamp: item.timestamp,
        cpu: item.cpu_percent,
        memory: item.memory_percent,
      }));

      setData(chartData);
    } catch (err) {
      console.error(`Failed to fetch history for container ${containerId}:`, err);
      setData([]);
    } finally {
      setLoading(false);
    }
  }, [containerId, localTimeRange]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  return (
    <div className="mt-4 pt-4 border-t border-gray-200">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2 text-sm text-gray-600">
          <BarChart3 className="h-4 w-4" />
          <span>Historical Performance - {containerName}</span>
        </div>
        <TimeRangeSelector selected={localTimeRange} onChange={setLocalTimeRange} />
      </div>
      <div className="h-48">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
          </div>
        ) : data.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-500">
            <div className="text-center">
              <Clock className="h-6 w-6 mx-auto mb-1 text-gray-400" />
              <p className="text-sm">No historical data</p>
            </div>
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis
                dataKey="time"
                tick={{ fontSize: 10, fill: '#6b7280' }}
                tickLine={false}
                axisLine={{ stroke: '#e5e7eb' }}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 10, fill: '#6b7280' }}
                tickLine={false}
                axisLine={{ stroke: '#e5e7eb' }}
                tickFormatter={(value) => `${value}%`}
                width={40}
              />
              <Tooltip content={<CustomTooltip />} />
              <Legend
                verticalAlign="top"
                height={24}
                iconType="line"
                wrapperStyle={{ fontSize: '11px' }}
              />
              <Line
                type="monotone"
                dataKey="cpu"
                stroke="#3b82f6"
                strokeWidth={2}
                dot={false}
                name="CPU"
              />
              <Line
                type="monotone"
                dataKey="memory"
                stroke="#8b5cf6"
                strokeWidth={2}
                dot={false}
                name="Memory"
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}

// Combined CPU & Memory Chart
function CombinedChart({
  data,
  timeRange,
  onTimeRangeChange,
  loading,
}: {
  data: any[];
  timeRange: string;
  onTimeRangeChange: (value: string) => void;
  loading: boolean;
}) {
  return (
    <Card className="p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="bg-gradient-to-br from-blue-100 to-purple-100 p-2 rounded-lg">
            <TrendingUp className="h-5 w-5 text-blue-600" />
          </div>
          <h3 className="text-lg font-medium text-gray-900">System Performance History</h3>
        </div>
        <TimeRangeSelector selected={timeRange} onChange={onTimeRangeChange} />
      </div>
      <div className="h-72">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : data.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-500">
            <div className="text-center">
              <Clock className="h-8 w-8 mx-auto mb-2 text-gray-400" />
              <p>No historical data available</p>
              <p className="text-sm text-gray-400">Data will appear once collected</p>
            </div>
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis
                dataKey="time"
                tick={{ fontSize: 11, fill: '#6b7280' }}
                tickLine={false}
                axisLine={{ stroke: '#e5e7eb' }}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 11, fill: '#6b7280' }}
                tickLine={false}
                axisLine={{ stroke: '#e5e7eb' }}
                tickFormatter={(value) => `${value}%`}
                width={45}
              />
              <Tooltip content={<CustomTooltip />} />
              <Legend
                verticalAlign="top"
                height={36}
                iconType="line"
                wrapperStyle={{ fontSize: '12px' }}
              />
              <Line
                type="monotone"
                dataKey="cpu"
                stroke="#3b82f6"
                strokeWidth={2}
                dot={false}
                name="CPU"
              />
              <Line
                type="monotone"
                dataKey="memory"
                stroke="#8b5cf6"
                strokeWidth={2}
                dot={false}
                name="Memory"
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </Card>
  );
}

export default function Monitoring() {
  const [historicalData, setHistoricalData] = useState<any[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [timeRange, setTimeRange] = useState('1h');
  const [expandedContainers, setExpandedContainers] = useState<Set<string>>(new Set());
  const [sseEnabled, setSseEnabled] = useState(true);

  // Use SSE for real-time metrics
  const {
    connected,
    error: sseError,
    systemMetrics,
    dockerMetrics,
    lastUpdate,
    reconnect,
  } = useMetricsSSE({
    interval: 5,
    enabled: sseEnabled,
    onData: (data) => {
      // Auto-expand running containers on first data
      if (data.docker?.containers && expandedContainers.size === 0) {
        const runningIds = data.docker.containers
          .filter((c) => c.state === 'running')
          .map((c) => c.id);
        setExpandedContainers(new Set(runningIds));
      }
    },
  });

  const toggleContainerExpand = (containerId: string) => {
    setExpandedContainers((prev) => {
      const next = new Set(prev);
      if (next.has(containerId)) {
        next.delete(containerId);
      } else {
        next.add(containerId);
      }
      return next;
    });
  };

  const fetchHistoricalData = useCallback(async () => {
    setHistoryLoading(true);
    try {
      const range = TIME_RANGES.find((r) => r.value === timeRange);
      const hours = range?.hours || 1;
      const from = new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
      const to = new Date().toISOString();

      // Determine resolution based on time range
      let resolution = 'raw';
      if (hours > 24) resolution = 'hourly';
      if (hours > 168) resolution = 'daily';

      const response = await api.get<HistoricalResponse>(
        `${API_ENDPOINTS.metricsHistory}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&resolution=${resolution}`
      );

      // Transform data for charts
      const chartData = (response.data || []).map((item) => ({
        time: formatChartTime(item.timestamp, timeRange),
        timestamp: item.timestamp,
        cpu: item.cpu_percent,
        memory: item.memory_percent,
      }));

      setHistoricalData(chartData);
    } catch (err) {
      console.error('Failed to fetch historical data:', err);
      setHistoricalData([]);
    } finally {
      setHistoryLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    fetchHistoricalData();
  }, [fetchHistoricalData]);

  // Also refresh historical data periodically when SSE is enabled
  useEffect(() => {
    if (!sseEnabled) return;

    const interval = setInterval(() => {
      fetchHistoricalData();
    }, 30000); // Refresh historical data every 30 seconds

    return () => clearInterval(interval);
  }, [sseEnabled, fetchHistoricalData]);

  // Show loading only on initial load (no SSE connection yet and no data)
  if (!connected && !systemMetrics) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Monitoring</h1>
          <p className="text-gray-600 mt-1">System and Docker container metrics</p>
        </div>
        <div className="flex items-center gap-4">
          {/* SSE Connection Status */}
          <div className="flex items-center gap-2">
            {connected ? (
              <span className="flex items-center gap-1.5 text-sm text-green-600">
                <Wifi className="h-4 w-4" />
                <span className="hidden sm:inline">Live</span>
              </span>
            ) : (
              <span className="flex items-center gap-1.5 text-sm text-gray-500">
                <WifiOff className="h-4 w-4" />
                <span className="hidden sm:inline">Disconnected</span>
              </span>
            )}
          </div>
          <label className="flex items-center gap-2 text-sm text-gray-600">
            <input
              type="checkbox"
              checked={sseEnabled}
              onChange={(e) => setSseEnabled(e.target.checked)}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            Real-time (5s)
          </label>
          <button
            onClick={() => {
              reconnect();
              fetchHistoricalData();
            }}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
        </div>
      </div>

      {sseError && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 flex items-center gap-3">
          <AlertTriangle className="h-5 w-5 text-red-500" />
          <span className="text-red-700">{sseError}</span>
          <button
            onClick={reconnect}
            className="ml-auto text-sm text-blue-600 hover:text-blue-700"
          >
            Retry
          </button>
        </div>
      )}

      {lastUpdate && (
        <p className="text-sm text-gray-500">
          Last updated: {lastUpdate.toLocaleTimeString()}
        </p>
      )}

      {/* System Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        {/* CPU */}
        <Card className="p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="bg-blue-100 p-3 rounded-lg">
                <Cpu className="h-6 w-6 text-blue-600" />
              </div>
              <div>
                <h3 className="font-medium text-gray-900">CPU</h3>
                <p className="text-xs text-gray-500">{systemMetrics?.cpu.cores} cores</p>
              </div>
            </div>
          </div>
          <div className="text-3xl font-bold text-gray-900 mb-2">
            {systemMetrics?.cpu.usage_percent.toFixed(1)}%
          </div>
          <ProgressBar value={systemMetrics?.cpu.usage_percent || 0} color="auto" />
          {systemMetrics?.load_avg && (
            <p className="text-xs text-gray-500 mt-2">
              Load: {systemMetrics.load_avg.map((l) => l.toFixed(2)).join(', ')}
            </p>
          )}
        </Card>

        {/* Memory */}
        <Card className="p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="bg-purple-100 p-3 rounded-lg">
                <MemoryStick className="h-6 w-6 text-purple-600" />
              </div>
              <div>
                <h3 className="font-medium text-gray-900">Memory</h3>
                <p className="text-xs text-gray-500">
                  {formatBytes(systemMetrics?.memory.total || 0)}
                </p>
              </div>
            </div>
          </div>
          <div className="text-3xl font-bold text-gray-900 mb-2">
            {systemMetrics?.memory.used_percent.toFixed(1)}%
          </div>
          <ProgressBar value={systemMetrics?.memory.used_percent || 0} color="auto" />
          <p className="text-xs text-gray-500 mt-2">
            {formatBytes(systemMetrics?.memory.used || 0)} /{' '}
            {formatBytes(systemMetrics?.memory.total || 0)}
          </p>
        </Card>

        {/* Uptime */}
        <Card className="p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="bg-green-100 p-3 rounded-lg">
                <Server className="h-6 w-6 text-green-600" />
              </div>
              <div>
                <h3 className="font-medium text-gray-900">Uptime</h3>
                <p className="text-xs text-gray-500">System</p>
              </div>
            </div>
          </div>
          <div className="text-3xl font-bold text-gray-900">
            {formatUptime(systemMetrics?.uptime || 0)}
          </div>
        </Card>

        {/* Docker Summary */}
        <Card className="p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="bg-orange-100 p-3 rounded-lg">
                <Container className="h-6 w-6 text-orange-600" />
              </div>
              <div>
                <h3 className="font-medium text-gray-900">Docker</h3>
                <p className="text-xs text-gray-500">
                  {dockerMetrics?.available ? dockerMetrics?.version : 'Not available'}
                </p>
              </div>
            </div>
          </div>
          {dockerMetrics?.available ? (
            <>
              <div className="text-3xl font-bold text-gray-900 mb-2">
                {dockerMetrics.summary.running}
                <span className="text-sm font-normal text-gray-500">
                  {' '}
                  / {dockerMetrics.summary.total} running
                </span>
              </div>
              <div className="flex gap-2 text-xs">
                <span className="px-2 py-1 bg-green-100 text-green-700 rounded">
                  {dockerMetrics.summary.running} running
                </span>
                <span className="px-2 py-1 bg-gray-100 text-gray-700 rounded">
                  {dockerMetrics.summary.stopped} stopped
                </span>
              </div>
            </>
          ) : (
            <p className="text-gray-500">Docker is not available</p>
          )}
        </Card>
      </div>

      {/* Combined Performance Chart */}
      <CombinedChart
        data={historicalData}
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
        loading={historyLoading}
      />

      {/* Individual Metric Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* CPU Chart */}
        <MetricChart
          title="CPU Usage"
          icon={Cpu}
          iconBgColor="bg-blue-100"
          iconColor="text-blue-600"
          data={historicalData}
          dataKey="cpu"
          color="#3b82f6"
          timeRange={timeRange}
          onTimeRangeChange={setTimeRange}
          loading={historyLoading}
          yAxisLabel="CPU"
          gradientId="cpuGradient"
        />

        {/* Memory Chart */}
        <MetricChart
          title="Memory Usage"
          icon={MemoryStick}
          iconBgColor="bg-purple-100"
          iconColor="text-purple-600"
          data={historicalData}
          dataKey="memory"
          color="#8b5cf6"
          timeRange={timeRange}
          onTimeRangeChange={setTimeRange}
          loading={historyLoading}
          yAxisLabel="Memory"
          gradientId="memoryGradient"
        />
      </div>

      {/* Disk Usage */}
      {systemMetrics?.disks && systemMetrics.disks.length > 0 && (
        <Card className="p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-yellow-100 p-3 rounded-lg">
              <HardDrive className="h-6 w-6 text-yellow-600" />
            </div>
            <h3 className="text-lg font-medium text-gray-900">Disk Usage</h3>
          </div>
          <div className="space-y-4">
            {systemMetrics.disks.map((disk, index) => (
              <div key={index} className="border rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <div>
                    <span className="font-medium text-gray-900">{disk.mountpoint}</span>
                    <span className="text-xs text-gray-500 ml-2">({disk.device})</span>
                  </div>
                  <span className="text-sm text-gray-600">
                    {formatBytes(disk.used)} / {formatBytes(disk.total)}
                  </span>
                </div>
                <ProgressBar value={disk.used_percent} color="auto" />
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Network Interfaces */}
      {systemMetrics?.network && systemMetrics.network.length > 0 && (
        <Card className="p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-teal-100 p-3 rounded-lg">
              <Network className="h-6 w-6 text-teal-600" />
            </div>
            <h3 className="text-lg font-medium text-gray-900">Network Interfaces</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-2 px-4 font-medium text-gray-600">Interface</th>
                  <th className="text-left py-2 px-4 font-medium text-gray-600">Status</th>
                  <th className="text-right py-2 px-4 font-medium text-gray-600">Received</th>
                  <th className="text-right py-2 px-4 font-medium text-gray-600">Sent</th>
                  <th className="text-right py-2 px-4 font-medium text-gray-600">Errors</th>
                </tr>
              </thead>
              <tbody>
                {systemMetrics.network.map((net, index) => (
                  <tr key={index} className="border-b last:border-0">
                    <td className="py-2 px-4 font-medium">{net.interface}</td>
                    <td className="py-2 px-4">
                      <span
                        className={`px-2 py-1 rounded text-xs ${
                          net.is_up
                            ? 'bg-green-100 text-green-700'
                            : 'bg-gray-100 text-gray-700'
                        }`}
                      >
                        {net.is_up ? 'Up' : 'Down'}
                      </span>
                    </td>
                    <td className="py-2 px-4 text-right">{formatBytes(net.bytes_recv)}</td>
                    <td className="py-2 px-4 text-right">{formatBytes(net.bytes_sent)}</td>
                    <td className="py-2 px-4 text-right text-gray-500">
                      {net.err_in + net.err_out > 0 ? (
                        <span className="text-red-600">
                          {net.err_in + net.err_out}
                        </span>
                      ) : (
                        0
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      {/* Docker Containers */}
      {dockerMetrics?.available && dockerMetrics.containers.length > 0 && (
        <Card className="p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-orange-100 p-3 rounded-lg">
              <Container className="h-6 w-6 text-orange-600" />
            </div>
            <h3 className="text-lg font-medium text-gray-900">Docker Containers</h3>
          </div>
          <div className="space-y-4">
            {/* Sort containers: running first, then paused, then exited/stopped */}
            {[...dockerMetrics.containers]
              .sort((a, b) => {
                const order = { running: 0, paused: 1, exited: 2, stopped: 2 };
                const aOrder = order[a.state as keyof typeof order] ?? 3;
                const bOrder = order[b.state as keyof typeof order] ?? 3;
                return aOrder - bOrder;
              })
              .map((container) => {
              const isExpanded = expandedContainers.has(container.id);
              return (
                <div
                  key={container.id}
                  className={`border rounded-lg p-4 ${
                    container.state === 'running' ? 'border-green-200 bg-green-50' : 'border-gray-200'
                  }`}
                >
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      {container.state === 'running' && (
                        <button
                          onClick={() => toggleContainerExpand(container.id)}
                          className="p-1 hover:bg-gray-200 rounded transition-colors"
                          title={isExpanded ? 'Hide history' : 'Show history'}
                        >
                          {isExpanded ? (
                            <ChevronUp className="h-4 w-4 text-gray-500" />
                          ) : (
                            <ChevronDown className="h-4 w-4 text-gray-500" />
                          )}
                        </button>
                      )}
                      <div>
                        <span className="font-medium text-gray-900">{container.name}</span>
                        <span className="text-xs text-gray-500 ml-2">({container.id})</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {container.state === 'running' && (
                        <button
                          onClick={() => toggleContainerExpand(container.id)}
                          className="flex items-center gap-1 px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 rounded transition-colors"
                        >
                          <BarChart3 className="h-3 w-3" />
                          History
                        </button>
                      )}
                      <span
                        className={`px-2 py-1 rounded text-xs ${
                          container.state === 'running'
                            ? 'bg-green-100 text-green-700'
                            : container.state === 'paused'
                            ? 'bg-yellow-100 text-yellow-700'
                            : 'bg-gray-100 text-gray-700'
                        }`}
                      >
                        {container.state}
                      </span>
                    </div>
                  </div>
                  <p className="text-xs text-gray-500 mb-3">{container.image}</p>

                  {container.state === 'running' && (
                    <>
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div>
                          <div className="flex items-center gap-1 text-xs text-gray-500 mb-1">
                            <Cpu className="h-3 w-3" /> CPU
                          </div>
                          <div className="text-sm font-medium">
                            {container.cpu.usage_percent.toFixed(1)}%
                          </div>
                          <ProgressBar
                            value={container.cpu.usage_percent}
                            color="auto"
                            showLabel={false}
                          />
                        </div>
                        <div>
                          <div className="flex items-center gap-1 text-xs text-gray-500 mb-1">
                            <MemoryStick className="h-3 w-3" /> Memory
                          </div>
                          <div className="text-sm font-medium">
                            {container.memory.used_percent.toFixed(1)}%
                          </div>
                          <ProgressBar
                            value={container.memory.used_percent}
                            color="auto"
                            showLabel={false}
                          />
                          <p className="text-xs text-gray-400 mt-1">
                            {formatBytes(container.memory.usage)} / {formatBytes(container.memory.limit)}
                          </p>
                        </div>
                        <div>
                          <div className="flex items-center gap-1 text-xs text-gray-500 mb-1">
                            <Network className="h-3 w-3" /> Network
                          </div>
                          <div className="text-xs">
                            <span className="text-green-600">
                              ↓ {formatBytes(container.network.rx_bytes)}
                            </span>
                            <span className="mx-1">/</span>
                            <span className="text-blue-600">
                              ↑ {formatBytes(container.network.tx_bytes)}
                            </span>
                          </div>
                        </div>
                        <div>
                          <div className="flex items-center gap-1 text-xs text-gray-500 mb-1">
                            <Activity className="h-3 w-3" /> Block I/O
                          </div>
                          <div className="text-xs">
                            <span className="text-green-600">
                              R {formatBytes(container.block_io.read_bytes)}
                            </span>
                            <span className="mx-1">/</span>
                            <span className="text-blue-600">
                              W {formatBytes(container.block_io.write_bytes)}
                            </span>
                          </div>
                        </div>
                      </div>

                      {/* Container History Chart */}
                      {isExpanded && (
                        <ContainerHistoryChart
                          containerId={container.id}
                          containerName={container.name}
                          timeRange={timeRange}
                        />
                      )}
                    </>
                  )}
                </div>
              );
            })}
          </div>
        </Card>
      )}
    </div>
  );
}
