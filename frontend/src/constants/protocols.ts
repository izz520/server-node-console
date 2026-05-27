export const SUPPORTED_PROTOCOLS = [
  "AnyTLS",
  "Any-reality",
  "Vless-xhttp-reality-vision-enc",
  "Vless-tcp-reality-vision",
  "Vless-xhttp-vision-enc",
  "Vless-ws-vision-enc",
  "Shadowsocks-2022",
  "Hysteria2",
  "Tuic",
  "Socks5",
  "Vmess-ws",
  "Argo 临时隧道",
  "Argo 固定隧道",
] as const;

export type SupportedProtocol = (typeof SUPPORTED_PROTOCOLS)[number];

export type InstallField =
  | "port"
  | "publicPort"
  | "uuid"
  | "realityDomain"
  | "cdnDomain"
  | "argoDomain"
  | "argoToken"
  | "namePrefix"
  | "remark";

export interface InstallProtocolFieldConfig {
  fields: InstallField[];
  requiredFields: InstallField[];
  uuidLabel?: string;
  uuidPlaceholder?: string;
}

const defaultInstallFieldConfig: InstallProtocolFieldConfig = {
  fields: ["port", "publicPort", "namePrefix", "remark"],
  requiredFields: [],
};

export const INSTALL_PROTOCOL_FIELD_CONFIG: Record<
  SupportedProtocol,
  InstallProtocolFieldConfig
> = {
  AnyTLS: {
    fields: ["port", "publicPort", "uuid", "namePrefix", "remark"],
    requiredFields: [],
  },
  "Any-reality": {
    fields: [
      "port",
      "publicPort",
      "uuid",
      "realityDomain",
      "namePrefix",
      "remark",
    ],
    requiredFields: [],
  },
  "Vless-xhttp-reality-vision-enc": {
    fields: [
      "port",
      "publicPort",
      "uuid",
      "realityDomain",
      "cdnDomain",
      "namePrefix",
      "remark",
    ],
    requiredFields: [],
  },
  "Vless-tcp-reality-vision": {
    fields: [
      "port",
      "publicPort",
      "uuid",
      "realityDomain",
      "namePrefix",
      "remark",
    ],
    requiredFields: [],
  },
  "Vless-xhttp-vision-enc": {
    fields: ["port", "publicPort", "uuid", "cdnDomain", "namePrefix", "remark"],
    requiredFields: [],
  },
  "Vless-ws-vision-enc": {
    fields: ["port", "publicPort", "uuid", "cdnDomain", "namePrefix", "remark"],
    requiredFields: [],
  },
  "Shadowsocks-2022": {
    fields: ["port", "publicPort", "uuid", "namePrefix", "remark"],
    requiredFields: [],
    uuidLabel: "Shadowsocks 2022 密钥",
    uuidPlaceholder: "留空则由后端生成 2022-blake3-aes-128-gcm 密钥",
  },
  Hysteria2: {
    fields: ["port", "publicPort", "namePrefix", "remark"],
    requiredFields: [],
  },
  Tuic: {
    fields: ["port", "publicPort", "namePrefix", "remark"],
    requiredFields: [],
  },
  Socks5: {
    fields: ["port", "publicPort", "namePrefix", "remark"],
    requiredFields: [],
  },
  "Vmess-ws": {
    fields: ["port", "publicPort", "uuid", "namePrefix", "remark"],
    requiredFields: [],
  },
  "Argo 临时隧道": {
    fields: ["port", "publicPort", "uuid", "cdnDomain", "namePrefix", "remark"],
    requiredFields: [],
  },
  "Argo 固定隧道": {
    fields: [
      "port",
      "publicPort",
      "uuid",
      "cdnDomain",
      "argoDomain",
      "argoToken",
      "namePrefix",
      "remark",
    ],
    requiredFields: ["argoDomain", "argoToken"],
  },
};

export function getInstallProtocolFieldConfig(protocol: string) {
  return (
    INSTALL_PROTOCOL_FIELD_CONFIG[protocol as SupportedProtocol] ??
    defaultInstallFieldConfig
  );
}

export const SUBSCRIPTION_FORMATS = [
  "sing-box",
  "Clash / Mihomo",
  "v2rayN",
  "Shadowrocket",
  "通用 Base64",
] as const;
