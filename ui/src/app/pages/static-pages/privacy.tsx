import { FlexBox } from '@/app/components/container/flex-box';

export function PrivacyPage() {
  return (
    <FlexBox>
      <div className="relative mx-auto pt-20 pb-24 max-w-6xl px-8 md:px-20 ">
        <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl">
          Privacy policy
        </h1>
        <p className="mt-4 text-base leading-7 ">
          Last updated on July 2, 2024
        </p>
        <main>
          <article className="relative mt-10 prose prose-base max-w-none! dark:prose-invert prose-slate">
            <p className="text-lg">
              Rapida.AI ("Rapida", "we", "us", or "our") respects the privacy of
              its customers ("Customers") and is committed to protecting the
              confidentiality and security of their data. This Privacy Policy
              describes the types of information we collect, how we use it, and
              with whom we share it. It also outlines your choices regarding
              your information and how to contact us with any questions.
            </p>
            <section id="data we collect automatically">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Data You Provide
                </h3>
                <div className="text-lg">
                  <h3 className="text-lg">1. Customer Account Information</h3>
                  When you create a Rapida account, we collect information such
                  as your company name, contact information (name, email
                  address, phone number), and billing details.
                </div>
                <div className="text-lg">
                  <h3 className="text-lg">2. Usage Data</h3> We may collect data
                  about your use of the Rapida platform, such as the types of
                  Generative AI models you integrate with, and the features you
                  use. This data is used to improve our services and provide
                  insights into customer behaviour.
                </div>
              </div>
            </section>
            <section id="information-we-collect">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Data We Collect Automatically
                </h3>
                <div className="text-lg">
                  <h3 className="text-lg">1. Log Data</h3> We may collect
                  standard log information when you use the Rapida platform,
                  such as your IP address, browser type, access times, and pages
                  viewed. This data is used for troubleshooting, security
                  purposes, and to understand how our platform is being used.
                </div>
              </div>
            </section>
            <section id="how-we-use-your-information">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  How We Use Your Information
                </h3>
                <ul className="list-disc list-inside mb-4 text-lg">
                  <li>Provide, operate, and maintain the Rapida platform.</li>
                  <li>
                    Improve and personalize your experience with the Rapida
                    platform.
                  </li>
                  <li>
                    Develop new features and functionalities for the Rapida
                    platform.
                  </li>
                  <li>
                    Send you important information about the Rapida platform,
                    such as updates, service changes, and security alerts.
                  </li>
                  <li>Comply with legal and regulatory requirements.</li>
                </ul>
              </div>
            </section>
            <section id="data-security">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Data Security</h3>
                <p className="text-lg">
                  We take reasonable steps to protect the information you
                  provide us from unauthorized access, disclosure, alteration,
                  or destruction. This includes implementing industry-standard
                  security measures such as encryption in transit and at rest,
                  access controls, and regular security audits.
                </p>
              </div>
            </section>
            <section id="sharing-your-information">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Sharing Your Information
                </h3>
                <p className="text-lg">
                  We will not share your information with any third party
                  without your consent, except in the following limited
                  circumstances:
                </p>
                <div className="text-lg">
                  <h3 className="text-lg">1. Service Providers</h3>
                  We may share your information with third-party service
                  providers who help us operate the Rapida platform, such as
                  data storage providers or customer support providers. These
                  service providers are contractually obligated to keep your
                  information confidential and secure.
                </div>

                <div className="text-lg">
                  <h3 className="text-lg">2. Legal Requirements</h3>
                  We may disclose your information if we are required to do so
                  by law, in response to a court order, or to comply with other
                  legal processes.
                </div>
              </div>
            </section>
            <section id="your-choices">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Your Choices</h3>
                <div className="text-lg">
                  <h3 className="text-lg">
                    1. Access and Update Your Information
                  </h3>
                  We may disclose your information if we are required to do so
                  by law, in response to a court order, or to comply with other
                  legal processes.
                </div>

                <div className="text-lg">
                  <h3 className="text-lg">2. Contact Us</h3> If you have any
                  questions about this Privacy Policy or your information,
                  please contact us at{' '}
                  <a href="mailto:support@rapida.ai" className="text-blue-500">
                    support@rapida.ai
                  </a>
                </div>
              </div>
            </section>
            <section id="international-transfers">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  International Transfers
                </h3>
                <div className="text-lg">
                  Rapida.AI is a global company, and your information may be
                  transferred to and processed in countries other than your own.
                  These countries may have different data protection laws than
                  your own. By using the Rapida platform, you consent to the
                  transfer of your information to these countries.
                </div>
              </div>
            </section>

            <section id="childrens-privacy">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Children's Privacy
                </h3>
                <div className="text-lg">
                  The Rapida platform is not intended for children under the age
                  of 18. We do not knowingly collect personal information from
                  children under 18.
                </div>
              </div>
            </section>

            <section id="changes-to-privacy-policy">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Changes to this Privacy Policy
                </h3>
                <div className="text-lg">
                  We may update this Privacy Policy from time to time. We will
                  notify you of any changes by posting the new Privacy Policy on
                  our website.
                </div>
              </div>
            </section>

            <section id="contact-us">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Contact Us</h3>
                <div className="text-lg">
                  If you have any questions about this Privacy Policy, please
                  contact us at{' '}
                  <a href="mailto:support@rapida.ai" className="text-blue-500">
                    support@rapida.ai
                  </a>
                </div>
              </div>
            </section>
          </article>
        </main>
      </div>
    </FlexBox>
  );
}
