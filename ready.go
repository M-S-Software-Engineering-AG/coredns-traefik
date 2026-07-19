package traefik

// Ready reports readiness to the ready plugin. The plugin is ready once every
// configured Traefik instance has completed its first refresh.
func (t *Traefik) Ready() bool {
	for _, inst := range t.Instances {
		inst.mutex.RLock()
		ready := inst.ready
		inst.mutex.RUnlock()
		if !ready {
			return false
		}
	}
	return true
}
