package disk

func (s *bundlerRepository) logInfo(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info(msg, args...)
	}
}

func (s *bundlerRepository) logWarn(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Warn(msg, args...)
	}
}

func (s *bundlerRepository) logError(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error(msg, args...)
	}
}
