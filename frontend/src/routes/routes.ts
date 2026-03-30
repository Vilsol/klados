import ClusterList from './ClusterList.svelte'
import ClusterOverview from './ClusterOverview.svelte'
import ResourceListPage from './ResourceListPage.svelte'
import ResourceDetailPage from './ResourceDetailPage.svelte'
import EventStreamPage from './EventStreamPage.svelte'
import PluginManagement from './PluginManagement.svelte'

export const routes = {
  '/': ClusterList,
  '/clusters': ClusterList,
  '/plugins': PluginManagement,
  '/c/:ctx/:gvr/:ns/:name': ResourceDetailPage,
  '/c/:ctx/events': EventStreamPage,
  '/c/:ctx/:gvr': ResourceListPage,
  '/c/:ctx': ClusterOverview,
  '/c/:ctx/*': ClusterOverview,
}
