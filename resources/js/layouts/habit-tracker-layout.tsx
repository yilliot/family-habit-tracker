import type { ReactNode } from 'react';
import Footer from '@/components/habit-tracker/footer';
import Masthead from '@/components/habit-tracker/masthead';
import { useFlashFeedback } from '@/components/habit-tracker/use-flash-feedback';

type Props = {
    children: ReactNode;
};

export default function HabitTrackerLayout({ children }: Props) {
    // Surfaces flash.success / flash.error as toasts (see fe-feedbackloop).
    useFlashFeedback();

    return (
        <div className="ht-root w-full">
            <div className="mx-auto w-full max-w-[1680px] px-4 pt-6 pb-12 md:px-6 lg:px-10">
                <div className="ht-page px-5 pt-8 pb-8 md:px-10 md:pt-10 md:pb-10 lg:px-14">
                    <Masthead />
                    <main>{children}</main>
                    <Footer />
                </div>
            </div>
        </div>
    );
}
