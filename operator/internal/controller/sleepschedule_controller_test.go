/*
Copyright 2024 Peter Valdez.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	snorlaxv1beta1 "moonbeam-nyc/snorlax/api/v1beta1"
	util "moonbeam-nyc/snorlax/internal/util"
)

var _ = Describe("SleepSchedule Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		sleepschedule := &snorlaxv1beta1.SleepSchedule{}

		BeforeEach(func() {
			By("Creating the custom resource for the Kind SleepSchedule")
			err := k8sClient.Get(ctx, typeNamespacedName, sleepschedule)
			if err != nil && errors.IsNotFound(err) {
				resource := &snorlaxv1beta1.SleepSchedule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// Base spec with example dailyWindow.
					// NOTE: Controller logic requires exactly one of dailyWindow or cronSchedule.
					Spec: snorlaxv1beta1.SleepScheduleSpec{
						DailyWindow: &snorlaxv1beta1.DailyWindow{
							WakeTime:  "9:00am",
							SleepTime: "5:00pm",
						},
						Timezone: "America/New_York",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance SleepSchedule")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			// Set the current time to 12:00am
			location, err := time.LoadLocation(sleepschedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 0, 0, 0, 0, location),
			}

			// Create the reconciler
			controllerReconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Reconciling the created resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(sleepschedule.Status.Awake).To(Equal(false))
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		It("should process DailyWindow configuration correctly", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			// Create a time stub
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 0, 0, 0, 0, location),
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule with DailyWindow")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())
			Expect(scheduleData).NotTo(BeNil())
			Expect(scheduleData.DailyWindow).NotTo(BeNil())
			Expect(scheduleData.CronSchedule).To(BeNil())

			By("Checking that the wake time is correct")
			Expect(scheduleData.DailyWindow.WakeTime).To(Equal(time.Date(2025, 1, 1, 9, 0, 0, 0, location)))

			By("Checking that the sleep time is correct")
			Expect(scheduleData.DailyWindow.SleepTime).To(Equal(time.Date(2025, 1, 1, 17, 0, 0, 0, location)))
		})

		It("should reconcile sleep schedule with DailyWindow to be awake", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Setting the current time to 9:01pm")
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 9, 1, 0, 0, location),
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule with DailyWindow")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the SleepSchedule should be awake")
			shouldBeAsleep, err := reconciler.shouldSleep(scheduleData)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldBeAsleep).To(BeFalse())
		})

		It("should reconcile sleep schedule with DailyWindow to be asleep", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Setting the current time to 5:01pm")
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 17, 1, 0, 0, location),
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule with DailyWindow")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the SleepSchedule should be asleep")
			shouldBeAsleep, err := reconciler.shouldSleep(scheduleData)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldBeAsleep).To(BeTrue())
		})

		It("should process CronSchedule configuration correctly", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Updating the SleepSchedule to use CronSchedule")
			existingSchedule.Spec.DailyWindow = nil
			existingSchedule.Spec.CronSchedule = &snorlaxv1beta1.CronSchedule{
				WakeSchedule:  "0 9 * * *",  // 9 AM every day
				SleepSchedule: "0 17 * * *", // 5 PM every day
			}
			Expect(k8sClient.Update(ctx, existingSchedule)).To(Succeed())

			// Create a time stub
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 0, 0, 0, 0, location),
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())
			Expect(scheduleData).NotTo(BeNil())
			Expect(scheduleData.CronSchedule).NotTo(BeNil())
			Expect(scheduleData.DailyWindow).To(BeNil())

			By("Checking that the wake schedule is correct")
			Expect(scheduleData.CronSchedule.WakeSchedule).To(Equal("0 9 * * *"))

			By("Checking that the sleep schedule is correct")
			Expect(scheduleData.CronSchedule.SleepSchedule).To(Equal("0 17 * * *"))
		})

		It("should reconcile sleep schedule with CronSchedule to be awake", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Updating the SleepSchedule to use CronSchedule with 9am wake and 5pm sleep every day")
			existingSchedule.Spec.DailyWindow = nil
			existingSchedule.Spec.CronSchedule = &snorlaxv1beta1.CronSchedule{
				WakeSchedule:  "0 9 * * *",  // 9am every day
				SleepSchedule: "0 17 * * *", // 5pm every day
			}
			Expect(k8sClient.Update(ctx, existingSchedule)).To(Succeed())

			By("Setting the current time to 9:01am")
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 9, 1, 0, 0, location),
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the SleepSchedule should be awake")
			shouldBeAsleep, err := reconciler.shouldSleep(scheduleData)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldBeAsleep).To(BeFalse())
		})

		It("should reconcile sleep schedule with CronSchedule to be asleep", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Updating the SleepSchedule with CronSchedule for 9am wake and 5pm sleep every day")
			existingSchedule.Spec.DailyWindow = nil
			existingSchedule.Spec.CronSchedule = &snorlaxv1beta1.CronSchedule{
				WakeSchedule:  "0 9 * * *",  // 9am every day
				SleepSchedule: "0 17 * * *", // 5pm every day
			}
			Expect(k8sClient.Update(ctx, existingSchedule)).To(Succeed())

			By("Setting the current time to 5:01pm")
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 17, 1, 0, 0, location),
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the SleepSchedule should be asleep")
			shouldBeAsleep, err := reconciler.shouldSleep(scheduleData)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldBeAsleep).To(BeTrue())
		})

		It("should reconcile sleep schedule with CronSchedule to be awake during weekdays", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Updating the SleepSchedule with CronSchedule for 12am wake every weekday and 12am sleep every weekend")
			existingSchedule.Spec.DailyWindow = nil
			existingSchedule.Spec.CronSchedule = &snorlaxv1beta1.CronSchedule{
				WakeSchedule:  "0 0 * * 1-5", // Every weekday at midnight
				SleepSchedule: "0 0 * * 6,7", // Every weekend at midnight
			}
			Expect(k8sClient.Update(ctx, existingSchedule)).To(Succeed())

			By("Setting the current time to 12:01am on a weekday")
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 1, 0, 1, 0, 0, location), // 12:01am on a Wednesday
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the SleepSchedule should be awake")
			shouldBeAsleep, err := reconciler.shouldSleep(scheduleData)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldBeAsleep).To(BeFalse())
		})

		It("should reconcile sleep schedule with CronSchedule to be asleep during weekends", func() {
			// Get the existing SleepSchedule created by BeforeEach
			existingSchedule := &snorlaxv1beta1.SleepSchedule{}
			err := k8sClient.Get(ctx, typeNamespacedName, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Updating the SleepSchedule with CronSchedule for 12am wake every weekday and 12am sleep every weekend")
			existingSchedule.Spec.DailyWindow = nil
			existingSchedule.Spec.CronSchedule = &snorlaxv1beta1.CronSchedule{
				WakeSchedule:  "0 0 * * 1-5", // Every weekday at midnight
				SleepSchedule: "0 0 * * 6,7", // Every weekend at midnight
			}
			Expect(k8sClient.Update(ctx, existingSchedule)).To(Succeed())

			By("Setting the current time to 12:01am on a weekend")
			location, err := time.LoadLocation(existingSchedule.Spec.Timezone)
			Expect(err).NotTo(HaveOccurred())
			timeStub := util.MockTime{
				Time: time.Date(2025, 1, 4, 0, 1, 0, 0, location), // 12:01am on a Saturday
			}

			// Create reconciler
			reconciler := NewReconciler(
				k8sClient,
				k8sClient.Scheme(),
				timeStub,
			)

			By("Processing the SleepSchedule")
			scheduleData, err := reconciler.ProcessSleepSchedule(ctx, existingSchedule)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the SleepSchedule should be asleep")
			shouldBeAsleep, err := reconciler.shouldSleep(scheduleData)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldBeAsleep).To(BeTrue())
		})
	})
})
