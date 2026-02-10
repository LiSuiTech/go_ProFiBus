# go_ProFiBus Dashboard

Vue 3 + TypeScript + Element Plus Dashboard for go_ProFiBus real-time data flow visualization and monitoring.

## Features

- 📊 **Real-time Pipeline Monitoring** - Monitor all data processing pipelines in real-time
- 🔄 **Live Data Flow Visualization** - See data flowing through your pipeline components
- 📈 **Performance Metrics** - Track throughput, latency, and error rates
- 🌐 **WebSocket Integration** - Real-time trace events via WebSocket
- 🎨 **Modern UI** - Built with Element Plus for a clean, responsive interface

## Tech Stack

- **Vue 3** - Progressive JavaScript framework
- **TypeScript** - Type-safe development
- **Vite** - Next generation frontend tooling
- **Pinia** - State management
- **Element Plus** - UI component library
- **ECharts** - Data visualization (planned)
- **D3.js** - Advanced visualization (planned)
- **Axios** - HTTP client

## Prerequisites

- Node.js 18+ and npm/yarn/pnpm
- Running go_ProFiBus backend API server (default: `http://localhost:8080`)

## Installation

```bash
# Install dependencies
npm install

# Or using yarn
yarn install

# Or using pnpm
pnpm install
```

## Development

```bash
# Start development server
npm run dev

# The dashboard will be available at http://localhost:3000
```

## Build

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

## Project Structure

```
web/dashboard/
├── src/
│   ├── components/          # Reusable Vue components
│   ├── views/               # Page components
│   │   ├── Dashboard.vue    # Main dashboard page
│   │   └── PipelineDetail.vue  # Pipeline detail page
│   ├── stores/              # Pinia stores
│   │   ├── pipeline.ts      # Pipeline state management
│   │   └── trace.ts         # Trace state and WebSocket
│   ├── services/            # API and WebSocket services
│   │   ├── api.ts           # REST API client
│   │   └── websocket.ts     # WebSocket client
│   ├── router/              # Vue Router configuration
│   ├── types/               # TypeScript type definitions
│   ├── App.vue              # Root component
│   └── main.ts              # Application entry point
├── index.html               # HTML template
├── vite.config.ts           # Vite configuration
├── tsconfig.json            # TypeScript configuration
└── package.json             # Project dependencies
```

## API Integration

The dashboard connects to the go_ProFiBus backend API:

### REST API Endpoints

- `GET /api/v1/pipelines` - Get all pipelines
- `GET /api/v1/pipelines/:id/topology` - Get pipeline topology
- `GET /api/v1/pipelines/:id/status` - Get pipeline status
- `POST /api/v1/pipelines/:id/start` - Start pipeline
- `POST /api/v1/pipelines/:id/stop` - Stop pipeline
- `GET /api/v1/pipelines/:id/metrics` - Get pipeline metrics
- `GET /api/v1/traces` - Get trace events
- `GET /api/v1/traces/stats` - Get trace statistics

### WebSocket

- `ws://localhost:8080/ws/trace` - Real-time trace events

## Configuration

### Backend API URL

In development mode, the dashboard proxies API requests to `http://localhost:8080`.

To change the backend URL, edit `vite.config.ts`:

```typescript
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://your-backend-url:8080',  // Change this
        changeOrigin: true,
      },
    },
  },
})
```

### WebSocket URL

The WebSocket URL is automatically determined based on the current host. For custom configuration, edit `src/services/websocket.ts`.

## Features Overview

### Dashboard Page

- View all pipelines
- Start/stop pipelines
- Monitor pipeline status in real-time
- View recent trace events

### Pipeline Detail Page

- Visualize pipeline topology
- View component flow (Source → Processors → Analyzers → Sinks)
- Monitor pipeline status and metrics
- Real-time performance statistics

## Development Tips

### Type Checking

```bash
npm run type-check
```

### Hot Module Replacement (HMR)

Vite provides fast HMR during development. Changes to Vue components, TypeScript files, and CSS will be reflected instantly.

### Debugging WebSocket

Open browser DevTools Console to see WebSocket connection logs:

```
[WebSocket] Connected
[WebSocket] Received trace event: {...}
```

## Troubleshooting

### Cannot connect to backend

1. Ensure the backend API server is running on `http://localhost:8080`
2. Check CORS configuration in the backend
3. Verify proxy configuration in `vite.config.ts`

### WebSocket connection fails

1. Ensure WebSocket endpoint is available at `ws://localhost:8080/ws/trace`
2. Check browser console for error messages
3. Verify firewall settings

### Build errors

1. Clear node_modules and reinstall:
   ```bash
   rm -rf node_modules package-lock.json
   npm install
   ```

2. Check Node.js version (requires 18+):
   ```bash
   node --version
   ```

## Roadmap

- [x] Basic dashboard layout
- [x] Pipeline list and management
- [x] Real-time WebSocket integration
- [x] Pipeline detail page with topology
- [ ] Advanced topology visualization with D3.js/Cytoscape.js
- [ ] Interactive performance charts with ECharts
- [ ] Trace timeline visualization
- [ ] Advanced filtering and search
- [ ] Export metrics and reports
- [ ] Dark mode support

## Contributing

Contributions are welcome! Please follow the existing code style and add tests for new features.

## License

MIT

## Links

- [Backend API Documentation](../../docs/API_EXAMPLES.md)
- [架构图与文档](../../docs/)（系统总体架构图、技术分层架构图等）
- [部署说明](../../docs/DEPLOYMENT.md)
