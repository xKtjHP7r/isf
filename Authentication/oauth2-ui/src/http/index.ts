import "regenerator-runtime/runtime";
import axios, {
  AxiosRequestConfig,
  AxiosResponse,
  AxiosInterceptorManager,
  CancelTokenStatic,
  AxiosInstance,
} from "axios";
import axiosRetry, { IAxiosRetryConfig } from "axios-retry";
import { replaceUrlParams } from "./util";
import RestAPI from "./api";
import Qs from "qs";

export interface OpenApiConfigBase extends Omit<
  AxiosRequestConfig,
  "baseURL" | "paramsSerializer"
> {
  hostname?: string;
  port?: number;
  token?: string;
  https?: boolean;
  "axios-retry"?: IAxiosRetryConfig;
  urlPrefix?: string;
}

export type API = RestAPI;

export type OpenApiHttpMethod<Path extends keyof API> = Path extends keyof API
  ? keyof API[Path]
  : never;

export type OpenApiRoute<
  Path extends keyof API,
  Method extends OpenApiHttpMethod<Path>,
> = API[Path][Method];

export type OpenApiRequestBody<
  Path extends keyof API,
  Method extends OpenApiHttpMethod<Path>,
> = OpenApiRoute<Path, Method> extends { body: infer T } ? T : void;

export type OpenApiResponseBody<
  Path extends keyof API,
  Method extends OpenApiHttpMethod<Path>,
> = OpenApiRoute<Path, Method> extends { response: infer T } ? T : never;

export interface OpenApiResponse<
  Path extends keyof API,
  Method extends OpenApiHttpMethod<Path>,
> extends Omit<AxiosResponse, "config" | "data"> {
  data: OpenApiResponseBody<Path, Method>;
  config: OpenApiRequestConfig<Path, Method> & {
    headers: Record<string, string>;
  };
}

export interface OpenApiRequestConfig<
  Path extends keyof API,
  Method extends OpenApiHttpMethod<Path>,
> extends OpenApiConfigBase {
  url?: Path;
  method?: Method;
  params?: { [key: string]: string | number };
  query?: { [key: string]: string | number | boolean | (string | number)[] };
  data?: OpenApiRequestBody<Path, Method>;
  urlPrefix?: string;
}

export type OpenApiPromise<
  Path extends keyof API,
  Method extends OpenApiHttpMethod<Path>,
> = Promise<OpenApiResponse<Path, Method>>;

export interface OpenApi {
  request<Path extends keyof API, Method extends OpenApiHttpMethod<Path>>(
    config: OpenApiRequestConfig<Path, Method>,
  ): OpenApiPromise<Path, Method>;
  get<Path extends keyof API, Method extends OpenApiHttpMethod<Path>>(
    url: Path,
    config?: OpenApiRequestConfig<Path, Method>,
  ): OpenApiPromise<Path, Method>;
  head<Path extends keyof API, Method extends OpenApiHttpMethod<Path>>(
    url: Path,
    config?: OpenApiRequestConfig<Path, Method>,
  ): OpenApiPromise<Path, Method>;
  delete<Path extends keyof API, Method extends OpenApiHttpMethod<Path>>(
    url: Path,
    config?: OpenApiRequestConfig<Path, Method>,
  ): OpenApiPromise<Path, Method>;
  post<Path extends keyof API, Method extends OpenApiHttpMethod<Path>>(
    url: Path,
    data?: OpenApiRequestBody<Path, Method>,
    config?: OpenApiRequestConfig<Path, Method>,
  ): OpenApiPromise<Path, Method>;
  put<Path extends keyof API, Method extends OpenApiHttpMethod<Path>>(
    url: Path,
    data?: OpenApiRequestBody<Path, Method>,
    config?: OpenApiRequestConfig<Path, Method>,
  ): OpenApiPromise<Path, Method>;
  patch<Path extends keyof API, Method extends OpenApiHttpMethod<Path>>(
    url: Path,
    data?: OpenApiRequestBody<Path, Method>,
    config?: OpenApiRequestConfig<Path, Method>,
  ): OpenApiPromise<Path, Method>;
}

function createOpenApiMethod<OpenApiMethod>(
  openApi: OpenApi,
  args: (keyof OpenApiRequestConfig<any, any>)[],
  currified: OpenApiRequestConfig<any, any> = {},
) {
  return ((...argv: any[]) => {
    const configs: any = {};

    args.forEach((arg, i) => {
      configs[arg] = argv[i];
    });

    return openApi.request({
      ...(argv[args.length] || {}),
      ...configs,
      ...currified,
    });
  }) as unknown as OpenApiMethod;
}

export interface OpenApiInterceptors {
  request: AxiosInterceptorManager<AxiosRequestConfig>;
  response: AxiosInterceptorManager<AxiosResponse>;
}

export class OpenApi {
  private _axios: AxiosInstance;
  public defaults: OpenApiConfigBase;

  constructor(config: OpenApiConfigBase = {}) {
    this._axios = axios.create();
    axiosRetry(this._axios as any, {
      retries: 0,
      retryCondition: () => false,
    });

    this.defaults = {
      https: true,
      hostname: "localhost",
      ...config,
    };

    this.request = this.request.bind(this);
    this.get = createOpenApiMethod<OpenApi["get"]>(this, ["url"], {
      method: "GET",
    });
    this.head = createOpenApiMethod<OpenApi["head"]>(this, ["url"], {
      method: "HEAD",
    });
    this.delete = createOpenApiMethod<OpenApi["delete"]>(this, ["url"], {
      method: "DELETE",
    });
    this.post = createOpenApiMethod<OpenApi["post"]>(this, ["url", "data"], {
      method: "POST",
    });
    this.put = createOpenApiMethod<OpenApi["put"]>(this, ["url", "data"], {
      method: "PUT",
    });
    this.patch = createOpenApiMethod<OpenApi["patch"]>(this, ["url", "data"], {
      method: "PATCH",
    });
  }

  public get interceptors(): OpenApiInterceptors {
    return this._axios.interceptors as OpenApiInterceptors;
  }

  public request<
    Path extends keyof API,
    Method extends OpenApiHttpMethod<Path>,
  >(config: OpenApiRequestConfig<Path, Method>): OpenApiPromise<Path, Method> {
    const combinedConfigs = { ...this.defaults, ...config };

    const {
      https,
      hostname,
      port,
      url,
      urlPrefix,
      method,
      headers,
      token,
      params,
      query,
      ...otherConfig
    } = combinedConfigs;

    const axiosUrl = replaceUrlParams(url as string, params);
    const pattern = new RegExp(/^\/wopi/g);
    const isWopiFront = pattern.test(axiosUrl);
    const url_prefix = !urlPrefix || urlPrefix === "/" ? "" : urlPrefix;

    const baseURL = `${https ? "https:" : "http:"}//${hostname}${port ? `:${port}` : ""}${url_prefix}${
      isWopiFront ? "" : "/api"
    }`;

    return this._axios.request({
      ...otherConfig,
      baseURL,
      url: axiosUrl,
      params: query,
      method,
      headers: {
        ...(headers || {}),
        Authorization: `Bearer ${token}`,
      },
      paramsSerializer: (params: any) =>
        Qs.stringify(params, { arrayFormat: "repeat" }),
    } as any) as unknown as OpenApiPromise<Path, Method>;
  }
}

function createInstance(config: OpenApiConfigBase = {}) {
  const instance = new OpenApi(config);
  return instance;
}

const openApi = createInstance();

export * from "./api";

export const CancelToken: CancelTokenStatic = axios.CancelToken;

export const create = createInstance;

export default openApi;
