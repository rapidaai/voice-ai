import { FlexBox } from '@/app/components/container/flex-box';

export function TermsPage() {
  return (
    <FlexBox>
      <div className="relative mx-auto pt-20 pb-24 max-w-6xl px-8 md:px-20">
        <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl">
          Terms of Service
        </h1>
        <p className="mt-4 text-base leading-7">Last updated on July 2, 2024</p>
        <main>
          <article className="relative mt-10 prose prose-base max-w-none! dark:prose-invert prose-slate">
            <p className="text-lg">
              These Terms of Service ("Terms") govern your access to and use of
              the Rapida.AI platform ("Platform"). By accessing or using the
              Platform, you agree to be bound by these Terms.
            </p>
            <section id="definitions">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Definitions</h3>
                <div className="text-lg">
                  <h3 className="text-lg">1. Customer</h3>
                  "Customer" means the business entity that has registered for
                  an account on the Platform.
                </div>
                <div className="text-lg">
                  <h3 className="text-lg">2. Content</h3>
                  "Content" means any data, text, code, images, or other
                  materials uploaded to or submitted through the Platform.
                </div>
                <div className="text-lg">
                  <h3 className="text-lg">3. Service</h3>
                  "Service" means the Rapida.AI platform, including all features
                  and functionalities.
                </div>
              </div>
            </section>
            <section id="account-creation">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Account Creation
                </h3>
                <div className="text-lg">
                  <p>
                    You must be a business entity to create an account on the
                    Platform.
                  </p>
                  <p>
                    You are responsible for maintaining the confidentiality of
                    your account credentials and for all activities that occur
                    under your account.
                  </p>
                  <p>
                    You agree to keep your account information accurate and
                    up-to-date.
                  </p>
                </div>
              </div>
            </section>
            <section id="use-of-the-service">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Use of the Service
                </h3>
                <div className="text-lg">
                  <p>
                    You may only use the Service for business purposes and in
                    accordance with these Terms.
                  </p>
                  <p>
                    You will not use the Service for any illegal or unauthorized
                    purpose.
                  </p>
                  <p>
                    You will not violate any applicable laws or regulations in
                    connection with your use of the Service.
                  </p>
                  <p>
                    You will not interfere with or disrupt the Service or the
                    servers or networks connected to the Service.
                  </p>
                  <p>
                    You are solely responsible for all Content you upload to or
                    submit through the Service.
                  </p>
                </div>
              </div>
            </section>
            <section id="intellectual-property">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Intellectual Property
                </h3>
                <div className="text-lg">
                  <p>
                    Rapida.AI owns all intellectual property rights in and to
                    the Platform.
                  </p>
                  <p>
                    You grant Rapida.AI a non-exclusive, worldwide, royalty-free
                    license to use, reproduce, modify, publish, and distribute
                    your Content in connection with the provision of the
                    Service.
                  </p>
                  <p>
                    You retain all ownership rights in your Content, but you
                    agree that Rapida.AI may use your Content for marketing or
                    promotional purposes.
                  </p>
                </div>
              </div>
            </section>
            <section id="third-party-services">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Third-Party Services
                </h3>
                <div className="text-lg">
                  <p>
                    The Platform may integrate with or link to third-party
                    services. You acknowledge that your use of these third-party
                    services is subject to their own terms and conditions.
                  </p>
                </div>
              </div>
            </section>
            <section id="disclaimers">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Disclaimers</h3>
                <div className="text-lg">
                  <p>
                    THE SERVICE IS PROVIDED "AS IS" AND "AS AVAILABLE" WITHOUT
                    WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT
                    LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
                    PARTICULAR PURPOSE, AND NON-INFRINGEMENT.
                  </p>
                </div>
              </div>
            </section>
            <section id="limitation-of-liability">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Limitation of Liability
                </h3>
                <div className="text-lg">
                  <p>
                    RAPIDA.AI SHALL NOT BE LIABLE FOR ANY DIRECT, INDIRECT,
                    INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES
                    ARISING OUT OF OR IN CONNECTION WITH YOUR USE OF THE
                    SERVICE, EVEN IF RAPIDA.AI HAS BEEN ADVISED OF THE
                    POSSIBILITY OF SUCH DAMAGES.
                  </p>
                </div>
              </div>
            </section>
            <section id="term-and-termination">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Term and Termination
                </h3>
                <div className="text-lg">
                  <p>
                    These Terms will remain in effect until terminated by either
                    you or Rapida.AI.
                  </p>
                  <p>
                    You may terminate these Terms at any time by discontinuing
                    your use of the Service.
                  </p>
                  <p>
                    Rapida.AI may terminate these Terms at any time for any
                    reason, with or without notice.
                  </p>
                </div>
              </div>
            </section>
            <section id="governing-law">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Governing Law</h3>
                <div className="text-lg">
                  <p>
                    These Terms will be governed by and construed in accordance
                    with the laws of the State of [Insert State], without regard
                    to its conflict of laws provisions.
                  </p>
                </div>
              </div>
            </section>
            <section id="dispute-resolution">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Dispute Resolution
                </h3>
                <div className="text-lg">
                  <p>
                    Any dispute arising out of or relating to these Terms will
                    be resolved by binding arbitration in accordance with the
                    rules of the American Arbitration Association. The
                    arbitration will be held in [Insert City, State].
                  </p>
                </div>
              </div>
            </section>
            <section id="entire-agreement">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">
                  Entire Agreement
                </h3>
                <div className="text-lg">
                  <p>
                    These Terms constitute the entire agreement between you and
                    Rapida.AI with respect to the subject matter hereof and
                    supersede all prior or contemporaneous communications,
                    representations, or agreements, whether oral or written.
                  </p>
                </div>
              </div>
            </section>
            <section id="severability">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Severability</h3>
                <div className="text-lg">
                  <p>
                    If any provision of these Terms is held to be invalid or
                    unenforceable, such provision will be struck and the
                    remaining provisions will remain in full force and effect.
                  </p>
                </div>
              </div>
            </section>
            <section id="waiver">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Waiver</h3>
                <div className="text-lg">
                  <p>
                    No waiver of any provision of these Terms will be deemed a
                    further or continuing waiver of such provision or any other
                    provision.
                  </p>
                </div>
              </div>
            </section>
            <section id="amendment">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Amendment</h3>
                <div className="text-lg">
                  <p>
                    Rapida.AI may amend these Terms at any time by posting the
                    amended Terms on the Platform. You are responsible for
                    periodically reviewing the Terms. Your continued use of the
                    Platform after the amended Terms are posted will be deemed
                    your acceptance of the amended Terms.
                  </p>
                </div>
              </div>
            </section>
            <section id="contact-us">
              <div className="mt-10 space-y-4">
                <h3 className="text-2xl font-semibold mb-2">Contact Us</h3>
                <div className="text-lg">
                  <p>
                    If you have any questions about these Terms, please contact
                    us at{' '}
                    <a
                      href="mailto:support@rapida.ai"
                      className="text-blue-500"
                    >
                      support@rapida.ai
                    </a>
                    .
                  </p>
                </div>
              </div>
            </section>
          </article>
        </main>
      </div>
    </FlexBox>
  );
}
