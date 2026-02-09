package route

// StandardRouteRepository: StandardRouteのリポジトリインターフェース
type StandardRouteRepository interface {
	Save(sr *StandardRoute) error
	FindByID(id StandardRouteID) (*StandardRoute, error)
}
