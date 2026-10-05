import {
    Github,
    Globe,
    Cloud,
    ShieldCheck,
    GitBranch,
    KeyRound,
    Fingerprint,
    Network,
    KeySquare,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { TFunction } from 'i18next'
import { ssoPresetLabel as ssoCustomPresetLabel } from '@/lib/labels'
import type { PresetField, SsoPresetCatalogEntry } from '@/types/sso'

const FIELD_KEY_BY_PRESET: Record<string, Record<string, string>> = {
    gitlab: { base_url: 'gitlab_base_url' },
    keycloak: { base_url: 'keycloak_base_url' },
    authentik: { base_url: 'authentik_base_url' },
}

export function ssoPresetLabel(preset: string | Pick<SsoPresetCatalogEntry, 'key' | 'label'>, t: TFunction): string {
    const key = typeof preset === 'string' ? preset : preset.key
    const fallback = typeof preset === 'string' ? preset : preset.label
    if (key === 'custom_oidc' || key === 'custom_saml') {
        return ssoCustomPresetLabel(t, key)
    }
    return fallback
}

/** The preset catalog is the server's, so a field it adds falls back to the label it sent. */
export function ssoFieldLabel(preset: string, field: PresetField, t: TFunction): string {
    const key = FIELD_KEY_BY_PRESET[preset]?.[field.key] ?? field.key
    const labels: Record<string, string> = {
        client_id: t('forms:sso.fields.client_id'),
        client_secret: t('forms:sso.fields.client_secret'),
        tenant: t('forms:sso.fields.tenant'),
        domain: t('forms:sso.fields.domain'),
        auth_server_id: t('forms:sso.fields.auth_server_id'),
        base_url: t('forms:sso.fields.base_url'),
        gitlab_base_url: t('forms:sso.fields.gitlab_base_url'),
        keycloak_base_url: t('forms:sso.fields.keycloak_base_url'),
        authentik_base_url: t('forms:sso.fields.authentik_base_url'),
        realm: t('forms:sso.fields.realm'),
        app_slug: t('forms:sso.fields.app_slug'),
        discovery_url: t('forms:sso.fields.discovery_url'),
        scopes: t('forms:sso.fields.scopes'),
        idp_entity_id: t('forms:sso.fields.idp_entity_id'),
        idp_sso_url: t('forms:sso.fields.idp_sso_url'),
        idp_x509_cert: t('forms:sso.fields.idp_x509_cert'),
        idp_x509_cert_standby: t('forms:sso.fields.idp_x509_cert_standby'),
    }
    return labels[key] ?? field.label
}

export function ssoFieldPlaceholder(field: PresetField, t: TFunction): string | undefined {
    const placeholders: Record<string, string> = {
        auth_server_id: t('forms:sso.placeholders.auth_server_id'),
        scopes: t('forms:sso.placeholders.scopes'),
    }
    return placeholders[field.key] ?? field.placeholder
}

// The frontend owns provider visuals; the backend only supplies the preset key.
const PRESET_ICONS: Record<string, LucideIcon> = {
    entra: Cloud,
    github: Github,
    google: Globe,
    okta: ShieldCheck,
    gitlab: GitBranch,
    keycloak: KeyRound,
    authentik: Fingerprint,
    custom_oidc: Network,
    custom_saml: KeySquare,
}

export function presetIcon(preset: string): LucideIcon {
    return PRESET_ICONS[preset] ?? KeyRound
}

export function isCustomPreset(preset: string): boolean {
    return preset === 'custom_oidc' || preset === 'custom_saml'
}
