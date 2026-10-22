// Sample SSO data for the stories of this folder.
import type { IdentityProvider, SsoPresetCatalogEntry } from '@/types/sso'

const secret = { key: 'client_secret', label: 'Client secret', type: 'password', secret: true, group: 'secrets' } as const

export const presets: SsoPresetCatalogEntry[] = [
    {
        key: 'github',
        label: 'GitHub',
        protocol: 'oauth2',
        fields: [{ key: 'client_id', label: 'Client ID', type: 'text', required: true, group: 'config' }, secret],
        default_claim_mappings: {},
    },
    {
        key: 'keycloak',
        label: 'Keycloak',
        protocol: 'oidc',
        fields: [
            { key: 'base_url', label: 'Base URL', type: 'url', required: true, group: 'config' },
            { key: 'realm', label: 'Realm', type: 'text', required: true, group: 'config', help: 'Usually master' },
            { key: 'client_id', label: 'Client ID', type: 'text', required: true, group: 'config' },
            secret,
        ],
        default_claim_mappings: {},
    },
    {
        key: 'custom_oidc',
        label: 'Custom OIDC',
        protocol: 'oidc',
        fields: [
            { key: 'discovery_url', label: 'Discovery URL', type: 'url', required: true, group: 'config' },
            { key: 'scopes', label: 'Scopes', type: 'text', group: 'config' },
        ],
        default_claim_mappings: {},
    },
    {
        key: 'custom_saml',
        label: 'Custom SAML',
        protocol: 'saml',
        fields: [
            { key: 'idp_entity_id', label: 'IdP entity ID', type: 'text', required: true, group: 'config' },
            { key: 'idp_sso_url', label: 'IdP SSO URL', type: 'url', required: true, group: 'config' },
            { key: 'idp_x509_cert', label: 'IdP certificate', type: 'textarea', required: true, group: 'config' },
        ],
        default_claim_mappings: {},
    },
]

export const provider = (overrides: Partial<IdentityProvider> = {}): IdentityProvider => ({
    id: 1,
    name: 'Company GitHub',
    slug: 'github',
    protocol: 'oauth2',
    preset: 'github',
    enabled: true,
    sortOrder: 0,
    config: { client_id: 'abc' },
    claimMappings: null,
    roleMapping: null,
    defaultRole: 'user',
    allowJit: true,
    syncRoleOnLogin: false,
    linkByEmail: true,
    hasClientSecret: true,
    metadataUrl: '',
    callbackUrl: 'https://savvy.example.com/api/auth/sso/github/callback',
    acsUrl: '',
    createdAt: '2030-01-01T00:00:00Z',
    updatedAt: '2030-01-01T00:00:00Z',
    ...overrides,
})
