import { Input } from 'antd';
import { useTranslation } from 'react-i18next';
import type { AllSetting } from '@/models/setting';
import { SettingListItem } from '@/components/ui';
import SecretInput from './SecretInput';

interface YooMoneyTabProps {
  allSetting: AllSetting;
  updateSetting: (patch: Partial<AllSetting>) => void;
}

export default function YooMoneyTab({ allSetting, updateSetting }: YooMoneyTabProps) {
  const { t } = useTranslation();

  return (
    <>
      <SettingListItem
        paddings="small"
        title={t('pages.settings.yoomoneyWallet')}
        description={t('pages.settings.yoomoneyWalletDesc')}
      >
        <Input
          value={allSetting.yoomoneyWallet}
          placeholder="41001..."
          onChange={(e) => updateSetting({ yoomoneyWallet: e.target.value })}
        />
      </SettingListItem>

      <SettingListItem
        paddings="small"
        title={t('pages.settings.yoomoneyClientId')}
        description={t('pages.settings.yoomoneyClientIdDesc')}
      >
        <Input
          value={allSetting.yoomoneyClientID}
          onChange={(e) => updateSetting({ yoomoneyClientID: e.target.value })}
        />
      </SettingListItem>

      <SettingListItem
        paddings="small"
        title={t('pages.settings.yoomoneyClientSecret')}
        description={
          allSetting.hasYooMoneyClientSecret && !allSetting.clearYooMoneyClientSecret
            ? t('pages.settings.yoomoneyClientSecretConfigured')
            : t('pages.settings.yoomoneyClientSecretDesc')
        }
      >
        <SecretInput
          value={allSetting.yoomoneyClientSecret}
          configured={allSetting.hasYooMoneyClientSecret}
          clearArmed={allSetting.clearYooMoneyClientSecret}
          placeholder={t('pages.settings.yoomoneyClientSecretPlaceholder')}
          onChange={(v) => updateSetting({ yoomoneyClientSecret: v })}
          onClearArmedChange={(armed) => updateSetting({ clearYooMoneyClientSecret: armed })}
        />
      </SettingListItem>

      <SettingListItem
        paddings="small"
        title={t('pages.settings.yoomoneyNotificationSecret')}
        description={
          allSetting.hasYooMoneyNotificationSecret && !allSetting.clearYooMoneyNotificationSecret
            ? t('pages.settings.yoomoneyNotificationSecretConfigured')
            : t('pages.settings.yoomoneyNotificationSecretDesc')
        }
      >
        <SecretInput
          value={allSetting.yoomoneyNotificationSecret}
          configured={allSetting.hasYooMoneyNotificationSecret}
          clearArmed={allSetting.clearYooMoneyNotificationSecret}
          placeholder={t('pages.settings.yoomoneyNotificationSecretPlaceholder')}
          onChange={(v) => updateSetting({ yoomoneyNotificationSecret: v })}
          onClearArmedChange={(armed) => updateSetting({ clearYooMoneyNotificationSecret: armed })}
        />
      </SettingListItem>
    </>
  );
}
