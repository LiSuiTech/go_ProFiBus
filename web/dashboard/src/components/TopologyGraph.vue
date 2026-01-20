<template>
  <div class="topology-graph">
    <svg ref="svgRef" :width="width" :height="height"></svg>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import * as d3 from 'd3'
import type { PipelineTopology } from '@/types'

interface Props {
  topology: PipelineTopology
  width?: number
  height?: number
}

const props = withDefaults(defineProps<Props>(), {
  width: 1000,
  height: 500,
})

const svgRef = ref<SVGSVGElement>()

interface Node {
  id: string
  name: string
  type: string
  x?: number
  y?: number
  fx?: number | null
  fy?: number | null
}

interface Link {
  source: string | Node
  target: string | Node
}

function renderGraph() {
  if (!svgRef.value || !props.topology) return

  // Clear previous content
  d3.select(svgRef.value).selectAll('*').remove()

  const svg = d3.select(svgRef.value)
  const width = props.width
  const height = props.height

  // Create nodes array
  const nodes: Node[] = []
  const links: Link[] = []

  // Add source node
  if (props.topology.components.source) {
    nodes.push({
      id: props.topology.components.source.id,
      name: props.topology.components.source.name,
      type: 'source',
    })
  }

  // Add processor nodes
  let previousId = props.topology.components.source?.id
  props.topology.components.processors?.forEach((proc, index) => {
    const nodeId = `processor-${index}`
    nodes.push({
      id: nodeId,
      name: proc.name,
      type: 'processor',
    })

    if (previousId) {
      links.push({ source: previousId, target: nodeId })
    }
    previousId = nodeId
  })

  // Add analyzer nodes
  props.topology.components.analyzers?.forEach((analyzer, index) => {
    const nodeId = `analyzer-${index}`
    nodes.push({
      id: nodeId,
      name: analyzer.name,
      type: 'analyzer',
    })

    if (previousId) {
      links.push({ source: previousId, target: nodeId })
    }
    previousId = nodeId
  })

  // Add sink nodes
  props.topology.components.sinks?.forEach((sink, index) => {
    const nodeId = `sink-${index}`
    nodes.push({
      id: nodeId,
      name: sink.name,
      type: 'sink',
    })

    if (previousId) {
      links.push({ source: previousId, target: nodeId })
    }
  })

  // Create force simulation
  const simulation = d3
    .forceSimulation<Node>(nodes)
    .force(
      'link',
      d3
        .forceLink<Node, Link>(links)
        .id((d) => d.id)
        .distance(150)
    )
    .force('charge', d3.forceManyBody().strength(-400))
    .force('center', d3.forceCenter(width / 2, height / 2))
    .force('collision', d3.forceCollide().radius(50))

  // Add arrow markers
  svg
    .append('defs')
    .selectAll('marker')
    .data(['arrow'])
    .enter()
    .append('marker')
    .attr('id', 'arrow')
    .attr('viewBox', '0 -5 10 10')
    .attr('refX', 35)
    .attr('refY', 0)
    .attr('markerWidth', 6)
    .attr('markerHeight', 6)
    .attr('orient', 'auto')
    .append('path')
    .attr('d', 'M0,-5L10,0L0,5')
    .attr('fill', '#999')

  // Create links
  const link = svg
    .append('g')
    .selectAll('line')
    .data(links)
    .enter()
    .append('line')
    .attr('stroke', '#999')
    .attr('stroke-width', 2)
    .attr('marker-end', 'url(#arrow)')

  // Create node groups
  const node = svg
    .append('g')
    .selectAll('g')
    .data(nodes)
    .enter()
    .append('g')
    .call(
      d3
        .drag<SVGGElement, Node>()
        .on('start', dragstarted)
        .on('drag', dragged)
        .on('end', dragended)
    )

  // Add circles to nodes
  node
    .append('circle')
    .attr('r', 30)
    .attr('fill', (d) => getNodeColor(d.type))
    .attr('stroke', '#fff')
    .attr('stroke-width', 3)

  // Add labels to nodes
  node
    .append('text')
    .text((d) => d.name)
    .attr('text-anchor', 'middle')
    .attr('dy', 50)
    .attr('font-size', '12px')
    .attr('font-weight', 'bold')

  // Add type labels
  node
    .append('text')
    .text((d) => d.type)
    .attr('text-anchor', 'middle')
    .attr('dy', 5)
    .attr('font-size', '10px')
    .attr('fill', '#fff')

  // Update positions on each tick
  simulation.on('tick', () => {
    link
      .attr('x1', (d: any) => d.source.x)
      .attr('y1', (d: any) => d.source.y)
      .attr('x2', (d: any) => d.target.x)
      .attr('y2', (d: any) => d.target.y)

    node.attr('transform', (d) => `translate(${d.x},${d.y})`)
  })

  // Drag functions
  function dragstarted(event: any, d: Node) {
    if (!event.active) simulation.alphaTarget(0.3).restart()
    d.fx = d.x
    d.fy = d.y
  }

  function dragged(event: any, d: Node) {
    d.fx = event.x
    d.fy = event.y
  }

  function dragended(event: any, d: Node) {
    if (!event.active) simulation.alphaTarget(0)
    d.fx = null
    d.fy = null
  }
}

function getNodeColor(type: string): string {
  const colors: Record<string, string> = {
    source: '#67C23A',
    processor: '#409EFF',
    analyzer: '#E6A23C',
    sink: '#F56C6C',
  }
  return colors[type] || '#909399'
}

onMounted(() => {
  renderGraph()
})

watch(
  () => props.topology,
  () => {
    renderGraph()
  },
  { deep: true }
)
</script>

<style scoped>
.topology-graph {
  width: 100%;
  display: flex;
  justify-content: center;
  background-color: #f5f7fa;
  border-radius: 8px;
  padding: 20px;
}

svg {
  background-color: white;
  border-radius: 4px;
}
</style>
