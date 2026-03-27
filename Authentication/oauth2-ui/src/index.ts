import express, { Request, Response, NextFunction } from "express";
import next from "next";
import cookieParser from "cookie-parser";
import bodyParser from "body-parser";
import csrf from "csurf";
import { createProxyMiddleware } from "http-proxy-middleware";
import signinHandler from "./controllers/signin";
import consentHandler from "./controllers/consent";
import signoutHandler from "./controllers/signout";
import userVerification from "./controllers/userVerification";
import { ErrorCode } from "./core/errorcode";
import getIFrameSize from "./controllers/iframeSize";

const port = parseInt(process.env.PORT || "4015", 10);

const env = process.env.NODE_ENV;
const debugHost = process.env.DEBUG_HOST;

const dev = env === "development";

const app = next({ dev });

const handle = app.getRequestHandler();

let server;

app
  .prepare()
  .then(async () => {
    server = express();

    // Set up the proxy
    if (dev) {
      server.use(
        "/api",
        createProxyMiddleware({
          target: `${debugHost}/api/`,
          pathRewrite: { "^/api": "/" },
          changeOrigin: true,
          secure: false,
        })
      );
      server.use(
        "/static",
        createProxyMiddleware({
          target: `${debugHost}/static/`,
          pathRewrite: { "^/static": "/" },
          changeOrigin: true,
          secure: false,
        })
      );
    }

    server.set("trust proxy", true);
    server.use(bodyParser.json());
    server.use(bodyParser.urlencoded({ extended: false }));
    server.use(cookieParser());

    server.use((err: any, _req: Request, res: Response, next: NextFunction) => {
      if (
        err instanceof SyntaxError ||
        err instanceof TypeError ||
        err instanceof RangeError ||
        err instanceof URIError
      ) {
        return res.status(400).json({
          code: ErrorCode.JSONFormatIllegal,
          message: "参数不合法",
        });
      }
      if (err instanceof ReferenceError) {
        return res.status(500).json({
          code: ErrorCode.InternalError,
          message: "内部错误",
        });
      }
      next(err);
    });

    const csrfProtection = csrf({
      cookie: {
        secure: true,
      },
    });
    server.use(csrfProtection as any);

    server.use(function (
      err: any,
      _req: Request,
      res: Response,
      next: NextFunction
    ) {
      if (err.code !== "EBADCSRFTOKEN") return next(err);
      res.status(403);
      res.json({
        code: 403041000,
        message: "csrf验证未通过。",
      });
    });

    server.post("/oauth2/signin", signinHandler);

    server.get("/oauth2/consent", consentHandler);

    server.get("/oauth2/signout", signoutHandler);

    server.get("/oauth2/userverification", userVerification);

    server.get("/oauth2/iframe-size", getIFrameSize);

    // server.get("/oauth2/static/*", csrfProtection, (req, res) => handle(req, res));

    // Default catch-all handler to allow Next.js to handle all other routes
    server.get("*", (req, res) => {
      return handle(req, res);
    });

    server
      .listen(port)
      .on("error", (err) => {
        throw err;
      })
      .on("listening", () => {
        console.log(`> Ready on port ${port} [${env}]`);
      });
  })
  .catch((err) => {
    console.log("An error occurred, unable to start the server");
    console.log(err);
  });
