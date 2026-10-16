package app

// ConfigWritable is whether the application changes the config file. It does not for now:
// the values in the file are shown in the admin panel but locked, and are edited in the
// file. Section.Update and config.Writable stay for when that is switched back on.
func (s *Services) ConfigWritable() bool { return false }
