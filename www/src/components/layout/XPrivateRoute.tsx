import * as React from "react";
import { FunctionComponent } from "react";
import { Redirect, Route, RouteProps } from "react-router-dom";

// A wrapper for <Route> that redirects to the login
// screen if you're not yet authenticated.
const XPrivateRoute: FunctionComponent<RouteProps & {
  authorized: boolean;
}> = ({ authorized, exact, path, component, children, ...props }) => {
  return (
    <Route
      exact={exact}
      path={path}

      render={({ location }) =>
        authorized ? (
          component ? (
            React.createElement(component, props, children)
          ) : (
              children
            )
        ) : (
            <Redirect
              to={{
                pathname: "/login",
                state: { from: location }
              }}
            />
          )
      }
    />
  );
};

export default XPrivateRoute;
